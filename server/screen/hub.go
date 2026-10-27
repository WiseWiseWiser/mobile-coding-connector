package screen

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"image/draw"
	"image/jpeg"
	"log"
	"os/exec"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// stopGrace is how long capture stays warm after the last viewer leaves. A tab
// switch closes the socket (an invisible page is not a client), and starting the
// capture child costs about a second, so a quick switch back should find the
// stream still running instead of paying that cold start again. Nothing is
// captured after this window: the stop timer is the only stop path.
var stopGrace = 10 * time.Second

// outMessage is a single WebSocket payload: JSON text or a binary frame.
type outMessage struct {
	text bool
	data []byte
}

// viewer is one connected browser tab or phone.
type viewer struct {
	id    string
	label string
	conn  *websocket.Conn
	send  chan outMessage
	// cursor holds at most one pending pointer update. Cursor reports are
	// absolute and cosmetic, so the newest one supersedes whatever is queued:
	// keeping them out of send means a fast pointer can never crowd out frames.
	cursor chan outMessage
	done   chan struct{}

	closeOnce sync.Once
	// needsKeyframe requests a full repaint (new viewer, or a dropped frame).
	needsKeyframe atomic.Bool
	// frames and dropped are published in the status snapshot.
	frames  atomic.Uint64
	dropped atomic.Uint64

	// limiter and lastDeny live behind Manager.mu.
	limiter  limiter
	lastDeny time.Time
}

func (v *viewer) close() {
	v.closeOnce.Do(func() {
		close(v.done)
		// A viewer created by a test has no socket; production ones do.
		if v.conn != nil {
			_ = v.conn.Close()
		}
	})
}

func (v *viewer) enqueue(msg outMessage) bool {
	select {
	case <-v.done:
		return false
	case v.send <- msg:
		return true
	default:
		return false
	}
}

// enqueueCursor queues a pointer update without ever blocking or dropping a
// frame: when the viewer is behind, the stale position is replaced.
func (v *viewer) enqueueCursor(msg outMessage) {
	select {
	case <-v.done:
		return
	default:
	}
	select {
	case v.cursor <- msg:
		return
	default:
	}
	select {
	case <-v.cursor:
	default:
	}
	select {
	case v.cursor <- msg:
	default:
	}
}

// write sends one message and closes the viewer when the connection fails.
func (v *viewer) write(msg outMessage) bool {
	kind := websocket.TextMessage
	if !msg.text {
		kind = websocket.BinaryMessage
	}
	_ = v.conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
	if err := v.conn.WriteMessage(kind, msg.data); err != nil {
		v.close()
		return false
	}
	return true
}

// writeLoop serializes all writes for one connection, as gorilla requires.
// Frames and JSON take priority: the cursor slot is only drained when nothing
// else is waiting, which keeps the newest pointer position on screen.
func (v *viewer) writeLoop() {
	for {
		select {
		case <-v.done:
			return
		case msg := <-v.send:
			if !v.write(msg) {
				return
			}
			continue
		default:
		}
		select {
		case <-v.done:
			return
		case msg := <-v.send:
			if !v.write(msg) {
				return
			}
		case msg := <-v.cursor:
			if !v.write(msg) {
				return
			}
		}
	}
}

// Manager owns the capture pipeline, the connected viewers and the control
// lock. A single instance is shared by every HTTP handler.
type Manager struct {
	mu  sync.Mutex
	cfg Config

	backend  backend
	displays []Display

	imgW, imgH int
	seq        uint32

	state   string
	lastErr string

	viewers     map[*viewer]struct{}
	holder      *viewer
	holderSince time.Time
	lastInput   time.Time
	lastAccept  time.Time

	tiler *tiler

	// hostLocked and inputOK mirror the two states that change what a viewer may
	// do without any action on their part — the host screen locking and the
	// input permission being granted. Both are edge triggered into a status
	// push so an open page recovers (and explains itself) without a reload.
	hostLocked bool
	inputOK    bool

	gen     int
	cancel  context.CancelFunc
	stopped chan struct{}
	stopAt  *time.Timer

	// decoded and published count the capture pipeline end to end.
	decoded   atomic.Uint64
	published atomic.Uint64

	// awake is the power-assertion child that keeps the host display on while
	// viewers are connected (nil when nothing holds it).
	awake *exec.Cmd
}

var (
	defaultOnce sync.Once
	defaultMgr  *Manager
)

// Default returns the process-wide manager.
func Default() *Manager {
	defaultOnce.Do(func() { defaultMgr = NewManager() })
	return defaultMgr
}

// Shutdown stops the shared manager's capture so its ffmpeg child does not
// outlive the server. A lingering child keeps the AVFoundation screen device
// busy and silently starves the next start.
func Shutdown() {
	Default().stop()
}

// NewManager builds a manager with persisted settings and the host backend.
func NewManager() *Manager {
	cfg := DefaultConfig()
	if loaded, err := LoadConfig(); err == nil && loaded != nil {
		cfg = *loaded
	}
	return &Manager{
		cfg:     cfg,
		backend: newBackend(),
		state:   StateStopped,
		viewers: make(map[*viewer]struct{}),
		tiler:   newTiler(TileSize, cfg.Quality),
	}
}

// Config returns a copy of the current settings.
func (m *Manager) Config() Config {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg
}

// UpdateConfig persists and applies new settings, restarting the capture
// pipeline when the geometry or frame rate changed.
func (m *Manager) UpdateConfig(cfg Config) error {
	cfg.normalize()
	if err := SaveConfig(&cfg); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	restart := cfg.MaxWidth != m.cfg.MaxWidth || cfg.FPS != m.cfg.FPS || cfg.Quality != m.cfg.Quality || cfg.Enabled != m.cfg.Enabled
	m.cfg = cfg
	if !cfg.AllowControl && m.holder != nil {
		m.releaseControlLocked("control disabled")
	}
	if !cfg.Enabled {
		m.stopCaptureLocked()
		m.state = StateStopped
	} else if restart && m.cancel != nil && len(m.viewers) > 0 {
		m.stopCaptureLocked()
		m.startCaptureLocked()
	}
	m.broadcastStatusLocked()
	return nil
}

// Snapshot reports the current state for GET /api/screen/status.
func (m *Manager) Snapshot() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.statusLocked()
}

func (m *Manager) statusLocked() Status {
	holder := ""
	if m.holder != nil {
		holder = m.holder.id
	}
	var since int64
	if !m.holderSince.IsZero() {
		since = m.holderSince.UnixMilli()
	}
	return Status{
		State:    m.state,
		Error:    m.lastErr,
		Platform: platformName(),
		Displays: m.displays,
		Capture: CaptureInfo{
			Tool:      captureToolName(),
			Width:     m.imgW,
			Height:    m.imgH,
			FPS:       m.cfg.FPS,
			IdleFPS:   m.cfg.IdleFPS,
			TileSize:  TileSize,
			Decoded:   m.decoded.Load(),
			Published: m.published.Load(),
		},
		Control: ControlInfo{Enabled: m.cfg.AllowControl, Holder: holder, Since: since},
		Input:   m.inputInfo(),
		Viewers: len(m.viewers),

		Sessions: m.sessionsLocked(),
	}
}

// sessionsLocked lists viewers in a stable order for the status snapshot.
func (m *Manager) sessionsLocked() []Session {
	out := make([]Session, 0, len(m.viewers))
	for v := range m.viewers {
		out = append(out, Session{
			ID:       v.id,
			Label:    v.label,
			Frames:   v.frames.Load(),
			Dropped:  v.dropped.Load(),
			Keyframe: v.needsKeyframe.Load(),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ---------------------------------------------------------------- lifecycle

func (m *Manager) addViewer(conn *websocket.Conn) *viewer {
	v := &viewer{
		id:     newViewerID(),
		conn:   conn,
		send:   make(chan outMessage, 16),
		cursor: make(chan outMessage, 1),
		done:   make(chan struct{}),
	}
	m.mu.Lock()
	if m.stopAt != nil {
		m.stopAt.Stop()
		m.stopAt = nil
	}
	m.viewers[v] = struct{}{}
	// Tiles are incremental, so a viewer that joins a live stream has missed
	// every pixel sent so far and would sit on an empty canvas until the screen
	// changes. Ask for a keyframe on the next captured frame.
	v.needsKeyframe.Store(true)
	if m.cancel == nil && m.cfg.Enabled {
		m.startCaptureLocked()
	}
	hello := map[string]any{
		"type":      "hello",
		"viewer_id": v.id,
		"label":     v.label,
		"status":    m.statusLocked(),
	}
	data, _ := json.Marshal(hello)
	v.enqueue(outMessage{text: true, data: data})
	m.replaySnapshotLocked(v)
	m.broadcastStatusLocked()
	m.mu.Unlock()
	return v
}

func (m *Manager) removeViewer(v *viewer) {
	v.close()
	m.mu.Lock()
	if _, ok := m.viewers[v]; !ok {
		m.mu.Unlock()
		return
	}
	delete(m.viewers, v)
	if m.holder == v {
		m.releaseControlLocked("viewer disconnected")
	}
	if len(m.viewers) == 0 {
		m.scheduleStopLocked()
	}
	m.broadcastStatusLocked()
	m.mu.Unlock()
}

func (m *Manager) scheduleStopLocked() {
	if m.stopAt != nil {
		m.stopAt.Stop()
	}
	var timer *time.Timer
	timer = time.AfterFunc(stopGrace, func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		// A viewer left again while this timer was already running and armed a
		// newer one. That newer timer owns the stop; clearing the handle here
		// would drop it and leak a capture nobody asked for.
		if m.stopAt != timer {
			return
		}
		m.stopAt = nil
		if len(m.viewers) == 0 {
			m.stopCaptureLocked()
		}
	})
	m.stopAt = timer
}

func (m *Manager) startCaptureLocked() {
	displays, err := m.backend.displays()
	if err != nil {
		m.state = StateError
		m.lastErr = err.Error()
		m.broadcastStatusLocked()
		return
	}
	m.displays = displays
	m.gen++
	gen := m.gen
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.stopped = make(chan struct{})
	m.state = StateStarting
	m.lastErr = ""
	m.lastAccept = time.Time{}
	m.tiler = newTiler(TileSize, m.cfg.Quality)
	// Seed the states the aux loop watches so its first tick only reports flips.
	m.hostLocked = hostLocked()
	m.inputOK = m.backend.info().Available
	cfg := m.cfg
	m.startAwakeLocked()
	log.Printf("[screen] starting capture: width<=%d@%dfps idle=%dfps quality=%d", cfg.MaxWidth, cfg.FPS, cfg.IdleFPS, cfg.Quality)
	go m.captureLoop(ctx, gen, cfg, m.stopped)
	go m.auxLoop(ctx, gen)
	m.broadcastStatusLocked()
}

func (m *Manager) stopCaptureLocked() {
	// Nobody will see the pixels any more; let the display sleep again.
	m.stopAwakeLocked()
	// Retire the generation before cancelling it. Cancelling only stops the next
	// read: a frame that was already past accept() is still being decoded, and
	// without this it publishes after the stop and marks the manager streaming
	// again — a status that claims a capture with no child, no loop and no
	// viewer. The same guard keeps the late failure from painting an error over
	// a clean stop.
	m.gen++
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.imgW, m.imgH = 0, 0
	if m.state != StateError {
		m.state = StateStopped
	}
}

func (m *Manager) stop() {
	m.mu.Lock()
	done := m.stopped
	m.stopCaptureLocked()
	m.state = StateStopped
	m.mu.Unlock()
	// Wait for the capture loop to tear down its ffmpeg child. Without this the
	// server can exit first, leaving an orphan that holds the screen device and
	// starves the next start.
	if done != nil {
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			log.Printf("[screen] capture loop still running after 3s, exiting anyway")
		}
	}
}

// ---------------------------------------------------------------- capture

func (m *Manager) captureLoop(ctx context.Context, gen int, cfg Config, done chan struct{}) {
	defer close(done)
	reapStale()
	src, err := startSourceFunc(ctx, cfg.MaxWidth, cfg.FPS, cfg.Quality)
	if err != nil {
		m.failCapture(gen, err)
		return
	}
	defer src.close()

	// A silent child process (blocked device or missing permission) never
	// returns from next(), so close it from under the reader instead of
	// waiting for an error that may never come.
	var firstFrame sync.Once
	gotFrame := make(chan struct{})
	var starved atomic.Bool
	watchdog := time.AfterFunc(captureStartTimeout, func() {
		select {
		case <-gotFrame:
		case <-ctx.Done():
		default:
			log.Printf("[screen] capture produced no frames in %s, stopping it", captureStartTimeout)
			starved.Store(true)
			src.close()
		}
	})
	defer watchdog.Stop()
	primed := false
	for {
		data, err := src.next(ctx)
		if err != nil {
			if errors.Is(err, errCaptureStopped) || ctx.Err() != nil {
				return
			}
			if !primed {
				// Either the watchdog closed a silent child or the child
				// failed outright; both mean we never saw a single frame.
				if starved.Load() {
					m.failCapture(gen, captureStartError(src))
				} else {
					m.failCapture(gen, captureFirstFrameError(src, err))
				}
				return
			}
			m.failCapture(gen, err)
			return
		}
		primed = true
		firstFrame.Do(func() { close(gotFrame) })
		if !m.accept(gen) {
			continue
		}
		img, err := jpeg.Decode(bytes.NewReader(data))
		if err != nil {
			continue
		}
		m.decoded.Add(1)
		m.publish(gen, toRGBA(img))
	}
}

func (m *Manager) failCapture(gen int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.gen != gen {
		return
	}
	m.state = StateError
	m.lastErr = err.Error()
	m.cancel = nil
	log.Printf("[screen] capture error: %v", err)
	m.broadcastStatusLocked()
}

// accept applies the active/idle frame-rate budget before paying for a JPEG
// decode. Skipping a frame here leaves the tiler state untouched, so nothing is
// lost: the next accepted frame still reports every changed tile.
func (m *Manager) accept(gen int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.gen != gen {
		return false
	}
	interval := time.Second / time.Duration(m.cfg.FPS)
	if time.Since(m.lastInput) > activeWindow {
		interval = time.Second / time.Duration(m.cfg.IdleFPS)
	}
	now := time.Now()
	if !m.lastAccept.IsZero() && now.Sub(m.lastAccept) < interval {
		return false
	}
	m.lastAccept = now
	return true
}

func toRGBA(img image.Image) *image.RGBA {
	b := img.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
	return rgba
}

func (m *Manager) publish(gen int, rgba *image.RGBA) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.gen != gen {
		return
	}
	w, h := rgba.Bounds().Dx(), rgba.Bounds().Dy()
	if w <= 0 || h <= 0 {
		return
	}
	if m.imgW != w || m.imgH != h {
		m.imgW, m.imgH = w, h
		m.tiler = newTiler(TileSize, m.cfg.Quality)
	}
	if m.needsKeyframeLocked() {
		m.tiler.reset()
	}
	tiles, full := m.tiler.diff(rgba)
	changedState := m.state != StateStreaming
	if changedState {
		m.state = StateStreaming
	}
	if len(tiles) == 0 && !full {
		if changedState {
			m.broadcastStatusLocked()
		}
		return
	}
	m.seq++
	m.published.Add(1)
	frame := encodeFrame(m.seq, uint32(time.Now().UnixMilli()), w, h, TileSize, TileSize, tiles, full)
	m.broadcastFrameLocked(frame)
	if changedState {
		m.broadcastStatusLocked()
	}
}

// replaySnapshotLocked sends the last published frame to v as a keyframe.
// Tiles are incremental, so without this a viewer that joins while the screen
// is static keeps an empty canvas: nothing changes, no frame is published, and
// the keyframe it asked for never comes.
func (m *Manager) replaySnapshotLocked(v *viewer) {
	if m.tiler == nil || !m.tiler.primed {
		return
	}
	tiles := m.tiler.snapshot()
	if len(tiles) == 0 {
		return
	}
	w, h := m.tiler.width, m.tiler.height
	// Reuse the current sequence: the client has no prior state, so a frame it
	// has already seen cannot be distinguished from this one anyway.
	frame := encodeFrame(m.seq, uint32(time.Now().UnixMilli()), w, h, TileSize, TileSize, tiles, true)
	if v.enqueue(outMessage{data: frame}) {
		v.needsKeyframe.Store(false)
	}
}

func (m *Manager) needsKeyframeLocked() bool {
	for v := range m.viewers {
		if v.needsKeyframe.Load() {
			return true
		}
	}
	return false
}

// broadcastFrameLocked fans a frame out to every viewer. A consumer that takes
// the frame has caught up and clears its keyframe request; one that drops it
// keeps the request, so the next publish repaints it in full. Clearing every
// viewer after the fan-out would erase exactly that retry and leave the slow
// viewer frozen on a stale canvas.
func (m *Manager) broadcastFrameLocked(frame []byte) {
	msg := outMessage{data: frame}
	for v := range m.viewers {
		if v.enqueue(msg) {
			v.frames.Add(1)
			v.needsKeyframe.Store(false)
			continue
		}
		v.dropped.Add(1)
		v.needsKeyframe.Store(true)
	}
}

func (m *Manager) broadcastStatusLocked() {
	payload := map[string]any{"type": "status", "status": m.statusLocked()}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	for v := range m.viewers {
		v.enqueue(outMessage{text: true, data: data})
	}
}

// ------------------------------------------------------------------ cursor

// auxLoop polls the host pointer at 30 Hz (so the browser cursor feels local)
// and, once a second, enforces the idle control timeout.
func (m *Manager) auxLoop(ctx context.Context, gen int) {
	cursor := time.NewTicker(time.Second / 30)
	house := time.NewTicker(time.Second)
	defer cursor.Stop()
	defer house.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-cursor.C:
			m.pollCursor(gen)
		case <-house.C:
			m.housekeeping(gen)
		}
	}
}

func (m *Manager) pollCursor(gen int) {
	x, y, ok := m.backend.pointerPos()
	if !ok {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.gen != gen || m.imgW <= 0 || len(m.displays) == 0 {
		return
	}
	ix, iy := hostToImage(m.displays[0], m.imgW, m.imgH, x, y)
	// The controller draws its own cursor from local mouse events (zero lag).
	// Everyone else sees the host pointer, which is what makes a session
	// watchable while somebody else drives.
	shown := encodeCursor(ix, iy, m.cursorShape(), 0, true)
	hidden := encodeCursor(ix, iy, m.cursorShape(), 0, false)
	for v := range m.viewers {
		if m.holder == v {
			v.enqueueCursor(outMessage{data: hidden})
		} else {
			v.enqueueCursor(outMessage{data: shown})
		}
	}
}

func (m *Manager) housekeeping(gen int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.gen != gen {
		return
	}
	if m.holder != nil && time.Since(m.lastInput) > controlIdleTimeout {
		m.releaseControlLocked("idle timeout")
	}
	locked := hostLocked()
	// Never leave a viewer holding control of a locked host: the next key would
	// land in the unlock prompt.
	if m.holder != nil && locked {
		m.releaseControlLocked("host screen locked")
	}
	// Both of these change what the viewer may do without the viewer doing
	// anything (the host locks, or the permission is granted while the page is
	// open), and a locked screen also explains a mirror that stopped changing.
	// Push a status on the edge instead of waiting for the next join or control
	// request to carry a fresh one — otherwise an open page keeps a dead
	// "Take control" button until it is reloaded.
	inputOK := m.backend.info().Available
	if locked != m.hostLocked || inputOK != m.inputOK {
		m.hostLocked = locked
		m.inputOK = inputOK
		log.Printf("[screen] host state changed: screen %s, input %s", lockWord(locked), accessWord(inputOK))
		m.broadcastStatusLocked()
	}
	// Capture is stopped by the grace timer armed when the last viewer left
	// (scheduleStopLocked), not here: the timer is the single stop path, so a
	// viewer that reconnects within the window cancels it instead of racing a
	// per-tick stop. Starting stays gated on addViewer.
}

// lockWord names the host screen state for logs.
func lockWord(locked bool) string {
	if locked {
		return "locked (capture shows the login window, input is refused)"
	}
	return "unlocked"
}

// accessWord names the input-injection permission for logs.
func accessWord(available bool) string {
	if available {
		return "available"
	}
	return "blocked by the Accessibility permission"
}

// ---------------------------------------------------------------- control

// controlIdleTimeout bounds how long a viewer keeps control without sending any
// input. It is generous on purpose: a viewer who takes control and then just
// watches the mirror should not lose the lock (and the "Take control" button
// state) after a minute of reading. Releasing on abandonment is still handled
// by the disconnect path, and the host-lock check below is the real safety net.
const controlIdleTimeout = 10 * time.Minute

func (m *Manager) requestControl(v *viewer) (bool, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.cfg.AllowControl {
		return false, "control is disabled on this host"
	}
	if info := m.backend.info(); !info.Available {
		return false, info.Reason
	}
	if hostLocked() {
		return false, "the host screen is locked"
	}
	if m.holder != nil && m.holder != v {
		return false, "another viewer is controlling this screen"
	}
	if m.holder == nil {
		m.holder = v
		m.holderSince = time.Now()
		log.Printf("[screen] control_granted viewer=%s", v.id)
	}
	m.lastInput = time.Now()
	payload, _ := json.Marshal(map[string]any{
		"type":      "control",
		"granted":   true,
		"viewer_id": v.id,
	})
	v.enqueue(outMessage{text: true, data: payload})
	m.broadcastStatusLocked()
	return true, ""
}

func (m *Manager) releaseControlLocked(reason string) {
	if m.holder == nil {
		return
	}
	id := m.holder.id
	m.holder = nil
	m.holderSince = time.Time{}
	m.backend.releaseAll()
	if reason != "" {
		log.Printf("[screen] control_released viewer=%s reason=%s", id, reason)
	}
	holder := ""
	if m.holder != nil {
		holder = m.holder.id
	}
	for v := range m.viewers {
		payload, _ := json.Marshal(map[string]any{
			"type":      "control",
			"granted":   false,
			"reason":    reason,
			"viewer_id": id,
			"holder":    holder,
		})
		v.enqueue(outMessage{text: true, data: payload})
	}
	m.broadcastStatusLocked()
}

// ControlHolderID reports the viewer currently holding control, if any.
func (m *Manager) ControlHolderID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.holder == nil {
		return ""
	}
	return m.holder.id
}

// ReleaseControl drops the current holder (used by the admin kill switch).
func (m *Manager) ReleaseControl(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.releaseControlLocked(reason)
}

func newViewerID() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "viewer"
	}
	return "v" + hex.EncodeToString(b[:])
}
