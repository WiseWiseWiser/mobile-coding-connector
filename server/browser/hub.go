package browser

import (
	"context"
	"encoding/json"
	"errors"
	"hash/fnv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Stream pacing. Screenshot capture costs ~35-45 ms round trip per frame, so
// there is no point asking for 30 fps: cast runs fast enough to watch a page
// load, and shot is a keepalive plus one frame per interaction.
const (
	// stopGrace keeps capture alive briefly after the last viewer leaves so a
	// reload or a network blip does not cost another window round trip.
	stopGrace = 10 * time.Second

	castIdleInterval   = 160 * time.Millisecond // 6 fps
	castActiveInterval = 83 * time.Millisecond  // 12 fps while the user is acting
	castActiveWindow   = 2 * time.Second
	shotInterval       = time.Second // 1 fps keepalive

	castKeepalive = time.Second
	shotKeepalive = 3 * time.Second

	captureTimeout = 20 * time.Second
	writeTimeout   = 15 * time.Second
)

// viewer is one connected browser page.
type viewer struct {
	conn *websocket.Conn
	wmu  sync.Mutex
}

// hub owns the capture goroutine and the viewer set. Capture only runs while
// somebody is watching, mirroring the desktop screen feature: nothing polls
// Chrome when nobody asked for frames.
type hub struct {
	m *Manager

	mu        sync.Mutex
	viewers   map[*viewer]struct{}
	stopTimer *time.Timer
	gen       int
	running   bool

	kickCh  chan struct{}
	inputCh chan InputEvent

	seq       uint32
	lastHash  uint64
	lastSent  time.Time
	lastInput time.Time
	frames    int
	lastMS    int
	lastSize  int
	lastErr   string
}

func newHub(m *Manager) *hub {
	h := &hub{
		m:       m,
		viewers: make(map[*viewer]struct{}),
		kickCh:  make(chan struct{}, 1),
		inputCh: make(chan InputEvent, 64),
	}
	go h.inputLoop()
	return h
}

// addViewer registers a viewer and starts capture when this is the first one.
func (h *hub) addViewer(v *viewer) {
	h.mu.Lock()
	if h.stopTimer != nil {
		h.stopTimer.Stop()
		h.stopTimer = nil
	}
	h.viewers[v] = struct{}{}
	start := !h.running
	if start {
		h.running = true
		h.gen++
		go h.loop(h.gen)
	}
	h.mu.Unlock()
	if start {
		h.kick()
	}
}

// removeViewer drops a viewer and schedules capture stop once the grace period
// passes. The timer is the only stop path, so a burst of disconnects cannot
// kill a stream that a new viewer just resumed.
func (h *hub) removeViewer(v *viewer) {
	h.mu.Lock()
	delete(h.viewers, v)
	empty := len(h.viewers) == 0
	if empty && h.stopTimer == nil && h.running {
		gen := h.gen
		h.stopTimer = time.AfterFunc(stopGrace, func() { h.stop(gen) })
	}
	h.mu.Unlock()
}

func (h *hub) stop(gen int) {
	h.mu.Lock()
	if gen != h.gen || len(h.viewers) > 0 {
		h.mu.Unlock()
		return
	}
	h.running = false
	h.gen++
	h.stopTimer = nil
	h.mu.Unlock()
}

// kick requests an immediate frame, e.g. right after a tap or a resize.
func (h *hub) kick() {
	select {
	case h.kickCh <- struct{}{}:
	default:
	}
}

func (h *hub) alive(gen int) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.running && h.gen == gen
}

func (h *hub) markInput() {
	h.mu.Lock()
	h.lastInput = time.Now()
	h.mu.Unlock()
}

func (h *hub) interval() time.Duration {
	h.mu.Lock()
	mode := h.m.mode
	active := time.Since(h.lastInput) < castActiveWindow
	h.mu.Unlock()
	if mode == ModeShot {
		return shotInterval
	}
	if active {
		return castActiveInterval
	}
	return castIdleInterval
}

// keepaliveLocked reports how long an unchanged frame may stay quiet. The
// caller must hold h.mu: publish() already needs it for the hash bookkeeping.
func (h *hub) keepaliveLocked() time.Duration {
	if h.m.mode == ModeShot {
		return shotKeepalive
	}
	return castKeepalive
}

// loop captures and publishes frames until the generation changes.
func (h *hub) loop(gen int) {
	failures := 0
	for {
		select {
		case <-h.kickCh:
		case <-time.After(h.interval()):
		}
		if !h.alive(gen) {
			return
		}
		// Keep the tab strip honest and follow the user's own tab switches in
		// Chrome. Throttled inside pollTabs, and bounded so a slow info job
		// cannot stall the frame loop for long.
		infoCtx, infoCancel := context.WithTimeout(context.Background(), 12*time.Second)
		h.m.pollTabs(infoCtx)
		infoCancel()
		if !h.alive(gen) {
			return
		}
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), captureTimeout)
		frame, err := h.m.Capture(ctx)
		cancel()
		if !h.alive(gen) {
			return
		}
		if err != nil {
			// Before Ensure has opened a content tab there is nothing to
			// capture yet: that is a normal warm-up state, not a failure.
			if errors.Is(err, errNoPageTab) {
				select {
				case <-time.After(300 * time.Millisecond):
				case <-h.kickCh:
				}
				continue
			}
			failures++
			h.m.setError(err)
			if failures == 1 {
				h.broadcastJSON(map[string]any{"type": "error", "message": err.Error()})
			}
			backoff := time.Duration(failures) * 400 * time.Millisecond
			if backoff > 3*time.Second {
				backoff = 3 * time.Second
			}
			select {
			case <-time.After(backoff):
			case <-h.kickCh:
			}
			continue
		}
		failures = 0
		h.publish(gen, frame, time.Since(start))
	}
}

// publish skips identical frames, keeping only a slow keepalive so the client
// can tell a still page from a dead socket.
func (h *hub) publish(gen int, jpeg []byte, took time.Duration) {
	if !h.alive(gen) {
		return
	}
	sum := hashBytes(jpeg)
	h.mu.Lock()
	same := sum == h.lastHash
	quiet := time.Since(h.lastSent)
	keepalive := h.keepaliveLocked()
	h.frames++
	h.lastMS = int(took / time.Millisecond)
	h.lastSize = len(jpeg)
	h.mu.Unlock()
	if same && quiet < keepalive {
		return
	}
	h.mu.Lock()
	h.lastHash = sum
	h.lastSent = time.Now()
	h.seq++
	seq := h.seq
	h.mu.Unlock()
	h.broadcast(encodeFrame(seq, time.Now().UnixMilli(), jpeg))
}

func hashBytes(b []byte) uint64 {
	f := fnv.New64a()
	_, _ = f.Write(b)
	return f.Sum64()
}

func (h *hub) broadcast(frame []byte) {
	h.mu.Lock()
	viewers := make([]*viewer, 0, len(h.viewers))
	for v := range h.viewers {
		viewers = append(viewers, v)
	}
	h.mu.Unlock()
	var dead []*viewer
	for _, v := range viewers {
		if err := v.writeFrame(frame); err != nil {
			dead = append(dead, v)
		}
	}
	for _, v := range dead {
		v.close()
		h.removeViewer(v)
	}
}

func (h *hub) broadcastJSON(payload any) {
	h.mu.Lock()
	viewers := make([]*viewer, 0, len(h.viewers))
	for v := range h.viewers {
		viewers = append(viewers, v)
	}
	h.mu.Unlock()
	for _, v := range viewers {
		_ = v.writeJSON(payload)
	}
}

func (h *hub) stats() CaptureStatus {
	h.mu.Lock()
	defer h.mu.Unlock()
	fps := 0
	if h.m.mode == ModeShot {
		fps = 1
	} else if time.Since(h.lastInput) < castActiveWindow {
		fps = int(time.Second / castActiveInterval)
	} else {
		fps = int(time.Second / castIdleInterval)
	}
	return CaptureStatus{
		Active:   h.running,
		Viewers:  len(h.viewers),
		FPS:      fps,
		Frames:   h.frames,
		LastMS:   h.lastMS,
		LastSize: h.lastSize,
	}
}

// enqueueInput queues one viewer action. Moves are coalesced by inputLoop so a
// fast drag cannot queue seconds of stale pointer positions.
func (h *hub) enqueueInput(ev InputEvent) {
	select {
	case h.inputCh <- ev:
	default:
		// Full: drop this event rather than block the socket reader. Discrete
		// events matter more than a lost move, so only move/up/tap get retried
		// after draining one stale move.
		select {
		case <-h.inputCh:
		default:
		}
		select {
		case h.inputCh <- ev:
		default:
		}
	}
}

// inputLoop applies input serially: each CDP call takes ~100 ms, and parallel
// calls would reorder a drag.
func (h *hub) inputLoop() {
	var pending *InputEvent
	for {
		var ev InputEvent
		if pending != nil {
			ev = *pending
			pending = nil
		} else {
			ev = <-h.inputCh
		}
		if ev.Kind == InputMove {
			for {
				var next InputEvent
				select {
				case next = <-h.inputCh:
				default:
				}
				if next.Kind == "" {
					break
				}
				if next.Kind == InputMove {
					ev = next
					continue
				}
				pending = &next
				break
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := h.m.SendInput(ctx, ev)
		cancel()
		h.markInput()
		if err != nil && !errors.Is(err, errNoPageTab) {
			h.m.setError(err)
			h.broadcastJSON(map[string]any{"type": "error", "message": err.Error()})
		}
		// Discrete actions change the page; ask for a frame now instead of
		// waiting out the cast interval.
		switch ev.Kind {
		case InputTap, InputUp, InputKey, InputText, InputWheel:
			h.kick()
		}
	}
}

func (v *viewer) writeFrame(frame []byte) error {
	v.wmu.Lock()
	defer v.wmu.Unlock()
	_ = v.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	return v.conn.WriteMessage(websocket.BinaryMessage, frame)
}

func (v *viewer) writeJSON(payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	v.wmu.Lock()
	defer v.wmu.Unlock()
	_ = v.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	return v.conn.WriteMessage(websocket.TextMessage, data)
}

func (v *viewer) close() {
	v.wmu.Lock()
	defer v.wmu.Unlock()
	_ = v.conn.Close()
}
