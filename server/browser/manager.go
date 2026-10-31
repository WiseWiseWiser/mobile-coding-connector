package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xhd2015/ai-critic/server/config"
)

// errNoPageTab means the session has no content tab yet: Ensure is still
// opening one, or the user closed it.
var errNoPageTab = errors.New("no page tab yet")

// Stream modes.
const (
	// ModeCast keeps a continuous frame stream: good for watching a page load,
	// a QR code rotate, or an animation.
	ModeCast = "cast"
	// ModeShot sends one frame per interaction plus a slow keepalive: good for
	// filling in a form where nothing moves between keystrokes.
	ModeShot = "shot"
)

// Window presets. The requested viewport is nominal; macOS refuses to shrink a
// Chrome window below ~500pt wide and caps it at the display work area, so the
// achieved viewport is always reported back to the client instead.
var presets = map[string]Viewport{
	"iphone": {Width: 390, Height: 844},
	"ipad":   {Width: 834, Height: 1194},
	"mac":    {Width: 1440, Height: 900},
}

// minMacWindowWidth is the narrowest window that may be remembered as the Mac
// preset's restore target. Phone and tablet presets are narrower than this, so
// an already-shrunk window can never be mistaken for the desktop size and
// restored later as 500x844.
const minMacWindowWidth = 900

// Preset names shown in the UI, in the order the switcher draws them.
var presetOrder = []string{"iphone", "ipad", "mac"}

// Bounds is a window rectangle in host points.
type Bounds struct {
	Left   int `json:"left"`
	Top    int `json:"top"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Viewport is the page's CSS viewport in CSS pixels.
type Viewport struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	DPR    float64 `json:"dpr,omitempty"`
}

// Config is the persisted browser settings (browser-config.json). Mode and
// preset are remembered so a reload comes back in the state the user chose, and
// the session is remembered so the page reopens on the window they were using.
// MacBounds keeps the last Mac-sized window's frame so the Mac preset can
// restore it across restarts.
type Config struct {
	Mode      string  `json:"mode"`
	Preset    string  `json:"preset"`
	SessionID string  `json:"session_id,omitempty"`
	MacBounds *Bounds `json:"mac_bounds,omitempty"`
}

func defaultConfig() Config {
	return Config{Mode: ModeCast, Preset: "mac", SessionID: SessionID}
}

// browserConfPath is resolved per call, not captured at init, so tests and
// alternate AI_CRITIC_HOME installs do not share one state file.
func browserConfPath() string { return filepath.Join(config.DataDir, "browser-config.json") }

func loadConfig() Config {
	cfg := defaultConfig()
	data, err := os.ReadFile(browserConfPath())
	if err != nil {
		return cfg
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return defaultConfig()
	}
	cfg.normalize()
	return cfg
}

func (c *Config) normalize() {
	if c.Mode != ModeCast && c.Mode != ModeShot {
		c.Mode = ModeCast
	}
	if _, ok := presets[c.Preset]; !ok {
		c.Preset = "mac"
	}
	if strings.TrimSpace(c.SessionID) == "" {
		c.SessionID = SessionID
	}
}

// saveSettings persists a preset/mode choice without letting a transient
// discovered-window id overwrite the session the next launch should reopen,
// or dropping the remembered Mac window bounds.
func saveSettings(mode, preset, sessionID string) {
	cfg := loadConfig()
	if isSessionlessSessionID(sessionID) {
		sessionID = cfg.SessionID
	}
	cfg.Mode = mode
	cfg.Preset = preset
	cfg.SessionID = sessionID
	_ = saveConfig(cfg)
}

func saveConfig(cfg Config) error {
	cfg.normalize()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(browserConfPath()); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(browserConfPath(), data, 0644)
}

// jobClient is the daemon surface the manager needs. Tests substitute a fake.
type jobClient interface {
	health(ctx context.Context) (*daemonHealth, error)
	sessions(ctx context.Context) ([]sessionRow, error)
	session(ctx context.Context, id string) (*sessionRow, error)
	createSession(ctx context.Context, id string) error
	createFreshSession(ctx context.Context, id string) error
	job(ctx context.Context, sessionID string, tabID int64, typ string, params any, timeout time.Duration) (json.RawMessage, error)
	// Direct rail (browser-agent 1.0.24+): jobs addressed by window id and a
	// sessionless window list. Implementations without direct support return
	// errors, and the manager falls back to the session track.
	jobDirect(ctx context.Context, windowID, tabID int64, typ string, params any, timeout time.Duration) (json.RawMessage, error)
	windowsDirect(ctx context.Context) (*windowsPayload, error)
}

// Manager owns the browser session: it makes sure the daemon, extension, and
// window exist, resizes that window, streams frames, and forwards input.
type Manager struct {
	mu       sync.Mutex
	ensureMu sync.Mutex
	startMu  sync.Mutex
	startup  string
	// attachMu serializes adoption of a discovered window. An attach is a
	// multi-step daemon conversation; two clicks must not create two /go pages.
	attachMu sync.Mutex

	d      jobClient
	port   int
	binary string
	opener func(ctx context.Context, url string) error
	// windowOpener opens a session page in a brand-new Chrome window (the
	// extension cannot create windows, so this goes through the host browser).
	windowOpener func(ctx context.Context, url string) error

	connected     bool
	daemonVersion string
	sessionID     string
	windowID      int64
	contentTabID  int64
	activeTabID   int64
	tabs          []TabInfo
	url           string
	extPath       string
	viewport      Viewport
	preset        string
	mode          string
	macBounds     Bounds
	macKnown      bool
	// presetApplied is false until the saved preset has been applied to the
	// current window. A restart or session switch starts a new window that may
	// still carry another preset's size, so the selector and the page must be
	// brought back in sync once.
	presetApplied bool
	lastErr       string
	ensuredAt     time.Time
	lastOpenAt    time.Time
	viewportAt    time.Time
	tabsAt        time.Time
	// newWindowFor marks the session whose page must open in a fresh Chrome
	// window (the + New window path); every other session page reuses the
	// window it already has.
	newWindowFor        string
	freshWindowRequired bool
	previousWindowIDs   map[int64]bool
	// readOnly is set when the daemon is too old for tab multiplexing, so the
	// rail can say why instead of firing jobs the extension would reject.
	readOnly bool
	// directSupported is true when the daemon accepts window_id addressing
	// without a session (direct-window-v1). Sessionless window ids
	// (sess-window-<id>) then route through the direct rail; everything else
	// keeps using sessions.
	directSupported bool

	hub *hub
}

var (
	defaultOnce sync.Once
	defaultMgr  *Manager
)

// Default returns the process-wide manager. The page, the API, and the capture
// loop must share one instance or two capture loops fight over the window.
func Default() *Manager {
	defaultOnce.Do(func() {
		defaultMgr = New(daemonPort())
	})
	return defaultMgr
}

// daemonPort honours BROWSER_AGENT_PORT so a test harness (or a second install)
// can talk to a daemon that is not on the port the extension manifest hardcodes.
// The extension still learns the port from /go?session=, so this is safe.
func daemonPort() int {
	raw := strings.TrimSpace(os.Getenv("BROWSER_AGENT_PORT"))
	if raw == "" {
		return DaemonPort
	}
	port, err := strconv.Atoi(raw)
	if err != nil || port <= 0 || port > 65535 {
		return DaemonPort
	}
	return port
}

// New builds a manager that talks to the browser-agent daemon on port.
func New(port int) *Manager {
	cfg := loadConfig()
	m := &Manager{
		d:         newDaemon(port),
		port:      port,
		binary:    findBinary(),
		preset:    cfg.Preset,
		mode:      cfg.Mode,
		sessionID: cfg.SessionID,
	}
	// The remembered Mac window survives restarts: the saved preset is often
	// a phone size, so the first ensure would otherwise shrink the window
	// before any Mac-sized bounds could be measured, leaving nothing to
	// restore the next time the person picks Mac.
	if cfg.MacBounds != nil {
		m.macBounds = *cfg.MacBounds
		m.macKnown = true
	}
	m.opener = openInChrome
	m.windowOpener = openNewChromeWindow
	m.hub = newHub(m)
	return m
}

// hub exposes the streaming side for HTTP handlers.
func (m *Manager) hubState() *hub { return m.hub }

// Mode reports the current stream mode.
func (m *Manager) Mode() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.mode
}

// Preset reports the current window preset.
func (m *Manager) Preset() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.preset
}

func (m *Manager) setError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err == nil {
		m.lastErr = ""
		return
	}
	m.lastErr = err.Error()
}

// clearErrorIf retires the status error when it is still exactly match — the
// failure a recovering caller is responsible for. Errors set since by other
// paths (preset apply, window ops) stay visible.
func (m *Manager) clearErrorIf(match string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lastErr == "" || m.lastErr != match {
		return false
	}
	m.lastErr = ""
	return true
}

func (m *Manager) contentTab() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.contentTabID
}

// sessionIDValue is the session every job is addressed to. Reading it per call
// (instead of passing it around) keeps the capture loop, the input path, and the
// HTTP handlers pointed at whatever session the user last switched to.
func (m *Manager) sessionIDValue() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessionIDValueLocked()
}

// job runs a job in the current session's Chrome window.
func (m *Manager) job(ctx context.Context, tabID int64, typ string, params any, timeout time.Duration) (json.RawMessage, error) {
	// Route through jobIn so the current session's id decides the rail: a
	// sessionless window id (direct mode) rides window addressing here too.
	return m.jobIn(ctx, "", tabID, typ, params, timeout)
}

// jobIn runs a job in an explicit session (the rail reads tabs of sessions the
// user is not currently watching).
//
// Sessionless window ids (sess-window-<id>) ride the direct rail when the
// daemon supports it: one hook here makes every existing caller — info,
// create_tab, eval, screenshot, tab ops — work against a plain Chrome window
// with no /go page. Any direct failure surfaces as a normal job error.
func (m *Manager) jobIn(ctx context.Context, sessionID string, tabID int64, typ string, params any, timeout time.Duration) (json.RawMessage, error) {
	if sessionID == "" {
		sessionID = m.sessionIDValue()
	}
	if windowID, ok := windowIDFromSessionlessID(sessionID); ok && m.directSupportedValue() {
		return m.d.jobDirect(ctx, windowID, tabID, typ, params, timeout)
	}
	return m.d.job(ctx, sessionID, tabID, typ, params, timeout)
}

// Ensure brings up the whole chain: daemon, session, extension, window, tab.
// It is safe to call repeatedly; the expensive steps are skipped when the
// session is already attached.
func (m *Manager) Ensure(ctx context.Context) error {
	m.ensureMu.Lock()
	defer m.ensureMu.Unlock()
	return m.ensureLocked(ctx, false)
}

// ensureLocked only launches Chrome when the caller represents an explicit user action.
func (m *Manager) ensureLocked(ctx context.Context, allowOpen bool) error {
	if err := m.ensureDaemon(ctx); err != nil {
		m.setError(err)
		return err
	}
	// Direct rail: a sessionless window id (or a dead current session on a
	// direct-capable daemon) can drive a plain Chrome window with no /go page.
	if m.directSupportedValue() {
		err := m.ensureDirectWindow(ctx, allowOpen)
		if err == nil {
			presetErr := m.ensurePreset(ctx)
			viewportErr := m.refreshViewport(ctx)
			m.mu.Lock()
			m.ensuredAt = time.Now()
			m.mu.Unlock()
			if presetErr != nil {
				m.setError(presetErr)
			} else {
				m.setError(viewportErr)
			}
			return nil
		}
		// A sessionless id has no session track to fall back to: the chain
		// below would open a /go anchor for a window id. Surface the direct
		// error instead.
		if _, sessionless := windowIDFromSessionlessID(m.sessionIDValue()); sessionless {
			m.setError(err)
			return err
		}
	}
	if err := m.ensureSession(ctx, allowOpen); err != nil {
		m.setError(err)
		return err
	}
	if err := m.ensureWindow(ctx); err != nil {
		m.setError(err)
		return err
	}
	// A saved preset outlives the window it was applied to. Re-apply it once
	// per window so the size shown in the switcher is the size being mirrored.
	presetErr := m.ensurePreset(ctx)
	viewportErr := m.refreshViewport(ctx)
	m.mu.Lock()
	m.ensuredAt = time.Now()
	m.mu.Unlock()
	// A failed resize does not make the session unusable; keep the error on the
	// status line but leave Ensure successful so the person can pick another
	// preset instead of being stuck behind the saved one. A measurement failure
	// is still useful to show when the resize itself worked.
	if presetErr != nil {
		m.setError(presetErr)
	} else {
		m.setError(viewportErr)
	}
	return nil
}

// ensurePreset applies the saved preset to a window that has not received it
// yet. It is best-effort: a resize failure is reported, and the next Ensure
// retries.
func (m *Manager) ensurePreset(ctx context.Context) error {
	m.mu.Lock()
	pending := !m.presetApplied
	preset := m.preset
	m.mu.Unlock()
	if !pending {
		return nil
	}
	if preset == "" {
		preset = "mac"
	}
	return m.applyPreset(ctx, preset)
}

func (m *Manager) ensureDaemon(ctx context.Context) error {
	h, err := m.d.health(ctx)
	if err == nil {
		m.recordDaemon(h)
		return nil
	}
	if m.binary == "" {
		return fmt.Errorf("browser-agent daemon is not running and the browser-agent CLI was not found (install it or set BROWSER_AGENT_BIN)")
	}
	if err := m.spawnDaemon(); err != nil {
		return fmt.Errorf("start browser-agent daemon: %w", err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for {
		h, err = m.d.health(ctx)
		if err == nil {
			m.recordDaemon(h)
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("browser-agent daemon did not become healthy: %w", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
}

// ensureSession registers the stable session id and waits for the extension to
// attach. The extension learns its session from the /go page, so that page has
// to stay open in the window we drive.
// recordDaemon remembers the daemon version and refuses tab multiplexing on a
// daemon that predates the cdp tab polyfills: a read-only rail is better than
// five buttons that each fail with "unknown method".
// versionLess compares dotted numeric versions ("1.0.20" < "1.0.21"). A
// non-numeric or empty version counts as too old: we would rather show a
// read-only rail than send a job the extension rejects.
func versionLess(have, want string) bool {
	if have == "" {
		return true
	}
	parse := func(v string) ([]int, bool) {
		parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
		out := make([]int, 0, len(parts))
		for _, p := range parts {
			if p == "" {
				out = append(out, 0)
				continue
			}
			i := 0
			for i < len(p) && p[i] >= '0' && p[i] <= '9' {
				i++
			}
			if i == 0 {
				return nil, false
			}
			n := 0
			for _, c := range p[:i] {
				n = n*10 + int(c-'0')
			}
			out = append(out, n)
		}
		return out, true
	}
	a, okA := parse(have)
	b, okB := parse(want)
	if !okA || !okB {
		return true
	}
	for i := 0; i < len(a) || i < len(b); i++ {
		var av, bv int
		if i < len(a) {
			av = a[i]
		}
		if i < len(b) {
			bv = b[i]
		}
		if av != bv {
			return av < bv
		}
	}
	return false
}

func (m *Manager) recordDaemon(h *daemonHealth) {
	m.mu.Lock()
	m.daemonVersion = h.Version
	m.readOnly = versionLess(h.Version, minTabDaemonVersion)
	m.directSupported = h.DirectSupported
	m.mu.Unlock()
}

// directSupportedValue reports whether the daemon advertised the direct rail.
func (m *Manager) directSupportedValue() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.directSupported
}

// DirectSupported exposes the daemon's direct-window capability to the API
// layer, which routes sessionless window switches accordingly.
func (m *Manager) DirectSupported() bool {
	return m.directSupportedValue()
}

func (m *Manager) ensureSession(ctx context.Context, allowOpen bool) error {
	sessionID := m.sessionIDValue()
	if err := m.d.createSession(ctx, sessionID); err != nil {
		return err
	}
	row, err := m.d.session(ctx, sessionID)
	if err != nil {
		return err
	}
	if row.Extension.Connected {
		m.mu.Lock()
		if m.freshWindowRequired {
			m.mu.Unlock()
			return fmt.Errorf("new session %q is already connected to an existing window", sessionID)
		}
		m.connected = true
		m.extPath = row.ExtensionInstallPath
		m.mu.Unlock()
		return nil
	}
	// A disconnected extension cannot vouch for its last page count. Passive
	// viewers never launch Chrome; an explicit action may open one anchor page.
	m.markDisconnected(sessionID)
	if !allowOpen {
		return errBrowserOffline
	}
	m.mu.Lock()
	shouldOpen := time.Since(m.lastOpenAt) > 10*time.Second
	if shouldOpen {
		m.lastOpenAt = time.Now()
	}
	m.mu.Unlock()
	if !shouldOpen {
		return fmt.Errorf("Chrome was just requested; wait before trying again")
	}
	if shouldOpen {
		url := m.sessionPageURL(sessionID)
		opener := m.opener
		if m.takeNewWindow(sessionID) {
			opener = m.windowOpener
		}
		if err := opener(ctx, url); err != nil {
			return fmt.Errorf("open the browser-agent session page: %w", err)
		}
		m.setStartupWaiting()
	}
	deadline := time.Now().Add(25 * time.Second)
	for {
		row, err = m.d.session(ctx, sessionID)
		if err == nil {
			m.mu.Lock()
			m.extPath = row.ExtensionInstallPath
			m.mu.Unlock()
		}
		if err == nil && row.Extension.Connected {
			m.mu.Lock()
			m.connected = true
			m.mu.Unlock()
			return nil
		}
		if time.Now().After(deadline) {
			if err != nil {
				return fmt.Errorf("waiting for the browser-agent extension: %v", err)
			}
			return fmt.Errorf("Chrome was requested, but its browser-agent extension did not connect; check Chrome and the installed extension before retrying")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// ensureWindow finds the Chrome window that holds the session page, picks the
// page tab to drive, and records the current bounds for the Mac preset.
// ensureDirectWindow drives a plain Chrome window through the direct rail, so
// ai-critic can mirror a window that has no /go page at all. It picks:
//  1. the window named by a sessionless id (sess-window-<id>),
//  2. the previously driven window,
//  3. the focused window,
//  4. the first normal window with a capturable tab.
//
// The picked window is recorded under its sessionless id, so the rest of the
// manager keeps its session-based shape (applyInfo, tabs, viewport, presets).
// A nil return means the direct rail took over; any error falls back to the
// session chain silently.
func (m *Manager) ensureDirectWindow(ctx context.Context, allowOpen bool) error {
	m.mu.Lock()
	sessionID := m.sessionIDValueLocked()
	freshWindowRequired := m.freshWindowRequired
	previousWindow := m.windowID
	m.mu.Unlock()
	if freshWindowRequired {
		// "+ New window" must produce a real new session; do not hijack it.
		return fmt.Errorf("direct rail skips new-window requests")
	}

	payload, err := m.d.windowsDirect(ctx)
	if err != nil {
		return fmt.Errorf("direct window list: %w", err)
	}
	if len(payload.Windows) == 0 {
		return fmt.Errorf("direct rail: no Chrome windows found")
	}

	var target *windowSnapshot
	if want, ok := windowIDFromSessionlessID(sessionID); ok {
		target = payload.windowByID(want)
		if target == nil {
			// The window this direct session was pinned to is closed (closed
			// by hand in Chrome). Fail over to another free window instead of
			// staying dead on the gone id; with no free window, keep the
			// "window gone" error and let the rail pick a window.
			target = pickFreeDirectWindow(payload)
			if target == nil {
				return fmt.Errorf("%w; the remaining Chrome windows are driven by other sessions", errWindowGone)
			}
		}
	} else {
		// A live session keeps its own track; do not steal it.
		if row, rowErr := m.d.session(ctx, sessionID); rowErr == nil && row.Extension.Connected {
			return fmt.Errorf("session %s is live; keeping the session track", sessionID)
		}
		target = pickDirectWindow(payload, previousWindow)
	}
	if target == nil {
		return fmt.Errorf("direct rail: no usable Chrome window")
	}
	directID := sessionlessSessionID(target.WindowID)

	m.mu.Lock()
	m.sessionID = directID
	m.connected = true
	m.windowID = target.WindowID
	m.lastErr = ""
	m.mu.Unlock()

	if err := m.ensureWindow(ctx); err != nil {
		m.mu.Lock()
		m.connected = false
		m.mu.Unlock()
		return err
	}
	return nil
}

// pickDirectWindow chooses a window to drive when the caller did not name one:
// the previous window if it still exists, else the focused window, else the
// first normal window that has a capturable tab, else the first normal window.
// pickFreeDirectWindow is the failover picker for a direct session whose window
// was closed by hand: the focused free window, else the first normal free
// window that has a capturable tab. "Free" means it hosts no session page —
// a window with a /go anchor belongs to a live session track and failover must
// not hijack it. With no free window it returns nil so the caller keeps the
// "window gone" error and the rail stays the way to pick.
func pickFreeDirectWindow(payload *windowsPayload) *windowSnapshot {
	var fallback *windowSnapshot
	for i := range payload.Windows {
		w := &payload.Windows[i]
		if w.Type != "normal" || hostsSessionPage(w) {
			continue
		}
		if w.Focused {
			return w
		}
		if fallback == nil {
			for _, t := range w.Tabs {
				if t.Capturable {
					fallback = w
					break
				}
			}
		}
	}
	return fallback
}

func hostsSessionPage(w *windowSnapshot) bool {
	for _, t := range w.Tabs {
		if t.Role == "session_page" || isSessionPageURL(t.URL) {
			return true
		}
	}
	return false
}

func pickDirectWindow(payload *windowsPayload, previousWindow int64) *windowSnapshot {
	if previousWindow != 0 {
		if w := payload.windowByID(previousWindow); w != nil && w.Type == "normal" {
			return w
		}
	}
	var fallback *windowSnapshot
	for i := range payload.Windows {
		w := &payload.Windows[i]
		if w.Type != "normal" {
			continue
		}
		if w.Focused {
			return w
		}
		if fallback == nil {
			for _, t := range w.Tabs {
				if t.Capturable {
					fallback = w
					break
				}
			}
		}
	}
	if fallback != nil {
		return fallback
	}
	for i := range payload.Windows {
		if payload.Windows[i].Type == "normal" {
			return &payload.Windows[i]
		}
	}
	return nil
}

func (m *Manager) ensureWindow(ctx context.Context) error {
	windowID, tabID, url, err := m.resolveTargets(ctx)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.windowID = windowID
	m.contentTabID = tabID
	m.url = url
	m.mu.Unlock()

	m.rememberMacBounds(ctx)
	return nil
}

// rememberMacBounds records the current window as the target the Mac preset
// restores. Only a genuinely desktop-wide window qualifies: after a restart the
// saved preset can be "iphone" while the window is still phone-sized, and a
// poisoned target would make every later Mac request restore the phone width.
// The frame is re-recorded whenever a Mac-sized window is seen, so the memory
// tracks the window the user actually keeps (and survives restarts on disk).
func (m *Manager) rememberMacBounds(ctx context.Context) {
	bounds, err := m.windowBounds(ctx)
	if err != nil || bounds.Width < minMacWindowWidth {
		return
	}
	persist := false
	m.mu.Lock()
	if !m.macKnown || m.macBounds != bounds {
		m.macBounds = bounds
		m.macKnown = true
		persist = true
	}
	m.mu.Unlock()
	if !persist {
		return
	}
	cfg := loadConfig()
	cfg.MacBounds = &bounds
	_ = saveConfig(cfg)
}

// resolveTargets asks the extension for the session window and its tabs. The
// session page is the registration anchor and must stay in the window; the tab
// it drives is the first tab that is not that page.
func (m *Manager) resolveTargets(ctx context.Context) (windowID, tabID int64, url string, err error) {
	sessionID := m.sessionIDValue()
	info, err := m.fetchInfo(ctx, sessionID)
	if err != nil {
		return 0, 0, "", err
	}
	windowID = info.WindowID
	if windowID == 0 {
		return 0, 0, "", fmt.Errorf("the browser-agent extension reported no window for this session")
	}
	m.mu.Lock()
	sameWindow := m.freshWindowRequired && m.previousWindowIDs[windowID]
	m.mu.Unlock()
	if sameWindow {
		return 0, 0, "", fmt.Errorf("Chrome attached the new session to the existing window instead of opening a new one")
	}
	tabID, url = chooseContentTab(info)
	if tabID == 0 {
		if _, sessionless := windowIDFromSessionlessID(sessionID); sessionless {
			// A just-opened direct window can still report its opener tab as
			// about:blank: the data: URL has not committed yet, while the
			// windows enumeration that adopted this window already saw the
			// committed page. Re-read briefly before creating a placeholder,
			// or the placeholder lands NEXT TO the opener tab.
			deadline := time.Now().Add(5 * time.Second)
			for {
				select {
				case <-ctx.Done():
					return 0, 0, "", ctx.Err()
				case <-time.After(300 * time.Millisecond):
				}
				retried, err := m.fetchInfo(ctx, sessionID)
				if err != nil {
					break // keep the first snapshot; the placeholder flow below handles it
				}
				if retried.WindowID == 0 {
					return 0, 0, "", fmt.Errorf("the browser-agent extension reported no window for this session")
				}
				info = retried
				if tabID, url = chooseContentTab(info); tabID != 0 {
					break
				}
				if time.Now().After(deadline) {
					break
				}
			}
		}
	}
	if tabID == 0 {
		created, err := m.createContentTab(ctx, windowID)
		if err != nil {
			return 0, 0, "", err
		}
		tabID = created
		// create_tab has already opened and activated this placeholder. Do not
		// publish the earlier, pre-creation info snapshot as if it were current.
		index := 0
		for i := range info.Tabs {
			info.Tabs[i].Active = false
			if info.Tabs[i].Index >= index {
				index = info.Tabs[i].Index + 1
			}
		}
		info.Tabs = append(info.Tabs, TabInfo{TabID: created, Index: index, Title: "ai-critic", URL: contentPlaceholderURL(), Active: true, Role: "user"})
		info.Active = created
	}
	m.applyInfo(sessionID, info)
	return windowID, tabID, url, nil
}

// sessionPageURL is the /go page that registers a session with the extension.
func (m *Manager) sessionPageURL(sessionID string) string {
	return fmt.Sprintf("http://127.0.0.1:%d/go?session=%s", m.port, url.QueryEscape(sessionID))
}

// takeNewWindow reports whether this session's page should open in a fresh
// Chrome window, consuming the flag so a later reopen reuses its window.
func (m *Manager) takeNewWindow(sessionID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.newWindowFor != sessionID {
		return false
	}
	m.newWindowFor = ""
	return true
}

// createContentTab opens a blank tab to drive, leaving the session page alone.
// contentPlaceholderHTML is what a freshly created content tab shows. Chrome's
// new-tab page and about:blank are not capturable by the extension, so a tab
// created just for ai-critic needs a real document.
const contentPlaceholderHTML = `<!doctype html><meta charset="utf-8">` +
	`<meta name="viewport" content="width=device-width,initial-scale=1">` +
	`<title>ai-critic</title>` +
	`<style>html,body{height:100%;margin:0;font:16px -apple-system,system-ui,sans-serif;` +
	`display:grid;place-items:center;background:#0f1115;color:#d7dbe0}` +
	`main{max-width:32rem;padding:2rem;text-align:center;line-height:1.5}</style>` +
	`<main><h1>Chrome is being driven by ai-critic</h1>` +
	`<p>Type a URL in the bar above the mirror and press Go.</p></main>`

func contentPlaceholderURL() string {
	return "data:text/html;charset=utf-8," + url.PathEscape(contentPlaceholderHTML)
}

// isContentURL reports whether a tab can be the page ai-critic drives.
func isContentURL(raw string) bool {
	switch {
	case raw == "" || raw == "about:blank":
		return false
	case strings.HasPrefix(raw, "http://"), strings.HasPrefix(raw, "https://"):
		return true
	case strings.HasPrefix(raw, "data:"):
		return true
	default:
		return false
	}
}

// isWebURL reports whether the driven tab may navigate there. The viewer only
// offers http(s); keeping the same line on the server means a stale client or a
// hand-written call cannot point the page at file:// or a browser-internal URL.
func isWebURL(raw string) bool {
	lower := strings.ToLower(raw)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

// reportableURL keeps internal placeholder documents out of the URL bar.
func reportableURL(raw string) string {
	if strings.HasPrefix(raw, "data:") {
		return ""
	}
	return raw
}

func (m *Manager) createContentTab(ctx context.Context, windowID int64) (int64, error) {
	return m.createTabIn(ctx, m.sessionIDValue(), windowID, contentPlaceholderURL())
}

// windowBounds reads the current window rectangle.
func (m *Manager) windowBounds(ctx context.Context) (Bounds, error) {
	m.mu.Lock()
	windowID := m.windowID
	m.mu.Unlock()
	params := map[string]any{"op": "get"}
	if windowID != 0 {
		params["window_id"] = windowID
	}
	data, err := m.job(ctx, m.contentTab(), "window", params, 15*time.Second)
	if err != nil {
		return Bounds{}, err
	}
	var out struct {
		WindowID int64  `json:"window_id"`
		Window   Bounds `json:"window"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return Bounds{}, fmt.Errorf("decode window: %w", err)
	}
	if out.WindowID != 0 {
		m.mu.Lock()
		m.windowID = out.WindowID
		m.mu.Unlock()
	}
	return out.Window, nil
}

// Navigate points the driven tab at url.
func (m *Manager) Navigate(ctx context.Context, rawURL string) error {
	url := normalizeURL(rawURL)
	if url == "" {
		return fmt.Errorf("empty url")
	}
	if !isWebURL(url) {
		return fmt.Errorf("unsupported url scheme: only http and https can be opened")
	}
	if err := m.Ensure(ctx); err != nil {
		return err
	}
	tab := m.contentTab()
	if _, err := m.job(ctx, tab, "cdp", map[string]any{
		"method": "Page.navigate",
		"params": map[string]any{"url": url},
	}, 20*time.Second); err != nil {
		m.setError(err)
		return err
	}
	m.mu.Lock()
	m.url = url
	m.mu.Unlock()
	// The viewport survives navigation, but the address bar does not; give the
	// renderer a moment, then re-measure so input maps onto the new page.
	select {
	case <-ctx.Done():
	case <-time.After(250 * time.Millisecond):
	}
	if err := m.refreshViewport(ctx); err != nil {
		m.setError(err)
	}
	m.hub.kick()
	return nil
}

var addressScheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)
var addressHostPort = regexp.MustCompile(`(?i)^[a-z0-9.-]+:\d+(?:[/?#]|$)`)

// NavigateTab targets an explicit session and content tab. It does not open,
// switch, or resize another window if the viewer's selection has gone stale.
func (m *Manager) NavigateTab(ctx context.Context, sessionID string, tabID int64, rawURL string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" || tabID <= 0 {
		return fmt.Errorf("session id and tab id are required")
	}
	raw := strings.TrimSpace(rawURL)
	if addressScheme.MatchString(raw) && !addressHostPort.MatchString(raw) && !isWebURL(raw) {
		return fmt.Errorf("unsupported url scheme: only http and https can be opened")
	}
	target := normalizeURL(raw)
	if target == "" {
		return fmt.Errorf("empty url")
	}
	if !isWebURL(target) {
		return fmt.Errorf("unsupported url scheme: only http and https can be opened")
	}
	parsed, parseErr := url.Parse(target)
	if parseErr != nil || parsed.Hostname() == "" {
		return fmt.Errorf("invalid address")
	}
	m.ensureMu.Lock()
	defer m.ensureMu.Unlock()
	if m.isReadOnly() {
		return errReadOnly
	}
	health, err := m.d.health(ctx)
	if err != nil {
		return fmt.Errorf("check browser-agent version: %w", err)
	}
	if !health.OK {
		return fmt.Errorf("browser-agent is not healthy")
	}
	m.recordDaemon(health)
	if m.isReadOnly() {
		return errReadOnly
	}
	if m.sessionIDValue() != sessionID {
		return fmt.Errorf("the current window changed; reopen the address editor")
	}
	// Direct rail: a sessionless window id has no registry row; the windows
	// list stands in for the extension's registration check below.
	if _, isSessionless := windowIDFromSessionlessID(sessionID); isSessionless && m.directSupportedValue() {
		info, err := m.fetchInfo(ctx, sessionID)
		if err != nil {
			return fmt.Errorf("check target window: %w", err)
		}
		tab := tabOf(info, tabID)
		if tab.TabID == 0 {
			return fmt.Errorf("the selected tab is no longer in that window")
		}
		if tab.Role == "session_page" || isSessionPageURL(tab.URL) || !isContentURL(tab.URL) {
			return fmt.Errorf("the selected tab cannot be navigated")
		}
		result, err := m.jobIn(ctx, sessionID, tabID, "cdp", map[string]any{"method": "Page.navigate", "params": map[string]any{"url": target}}, 20*time.Second)
		if err != nil {
			return err
		}
		return m.applyNavigation(ctx, sessionID, tabID, target, info, result)
	}
	row, err := m.d.session(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("check target session: %w", err)
	}
	if !row.Extension.Connected {
		return fmt.Errorf("target browser session is disconnected")
	}
	info, err := m.fetchInfo(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("check target tab: %w", err)
	}
	tab := tabOf(info, tabID)
	if tab.TabID == 0 {
		return fmt.Errorf("the selected tab is no longer in that window")
	}
	if tab.Role == "session_page" || isSessionPageURL(tab.URL) || !isContentURL(tab.URL) {
		return fmt.Errorf("the selected tab cannot be navigated")
	}
	result, err := m.jobIn(ctx, sessionID, tabID, "cdp", map[string]any{"method": "Page.navigate", "params": map[string]any{"url": target}}, 20*time.Second)
	if err != nil {
		return err
	}
	return m.applyNavigation(ctx, sessionID, tabID, target, info, result)
}

// applyNavigation decodes a Page.navigate job outcome and refreshes the URL
// bar when the navigation hit the window the viewer is watching.
func (m *Manager) applyNavigation(ctx context.Context, sessionID string, tabID int64, target string, info *sessionInfo, result json.RawMessage) error {
	var outcome struct {
		Result struct {
			ErrorText string `json:"errorText"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &outcome); err != nil {
		return fmt.Errorf("decode navigation result: %w", err)
	}
	if outcome.Result.ErrorText != "" {
		return fmt.Errorf("navigation failed: %s", outcome.Result.ErrorText)
	}
	m.mu.Lock()
	current := m.sessionIDValueLocked() == sessionID && m.contentTabID == tabID
	if current {
		m.url = target
		m.viewportAt = time.Time{}
	}
	m.mu.Unlock()
	if current {
		m.hub.kick()
		m.broadcastStatus()
	}
	return nil
}

// CurrentURL reports the driven page URL.
func (m *Manager) CurrentURL() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.url
}

func normalizeURL(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if strings.Contains(s, "://") || strings.HasPrefix(s, "about:") {
		return s
	}
	if strings.HasPrefix(s, "localhost:") || strings.HasPrefix(s, "127.0.0.1:") {
		return "http://" + s
	}
	// A bare host is a URL; a bare word is a search query.
	if strings.Contains(strings.SplitN(s, "/", 2)[0], ".") {
		return "https://" + s
	}
	return "https://www.google.com/search?q=" + url.QueryEscape(s)
}

// SetPreset resizes the window so its viewport is as close to the preset as the
// OS allows, then reports the achieved size. macOS clamps Chrome to a minimum
// width and to the display work area, so this never lies about the result.
func (m *Manager) SetPreset(ctx context.Context, name string) error {
	if _, ok := presets[name]; !ok {
		return fmt.Errorf("unknown preset %q", name)
	}
	if err := m.Ensure(ctx); err != nil {
		return err
	}
	return m.applyPreset(ctx, name)
}

// applyPreset resizes the current window and records the choice. It does not
// call Ensure, so Ensure can use it to apply a saved preset without recursing.
func (m *Manager) applyPreset(ctx context.Context, name string) error {
	preset, ok := presets[name]
	if !ok {
		return fmt.Errorf("unknown preset %q", name)
	}
	// Remember the Mac-sized window before the first shrink so the Mac preset
	// can restore it exactly instead of guessing 1440x900.
	if name != "mac" {
		m.rememberMacBounds(ctx)
	}
	m.mu.Lock()
	macKnown := m.macKnown
	macBounds := m.macBounds
	windowID := m.windowID
	m.mu.Unlock()
	if name == "mac" && macKnown {
		if _, err := m.job(ctx, m.contentTab(), "window", map[string]any{
			"op":        "update",
			"window_id": windowID,
			"left":      macBounds.Left,
			"top":       macBounds.Top,
			"width":     macBounds.Width,
			"height":    macBounds.Height,
			"state":     "normal",
		}, 20*time.Second); err != nil {
			m.setError(err)
			return err
		}
	} else {
		data, err := m.job(ctx, m.contentTab(), "window", map[string]any{
			"op":              "resize",
			"window_id":       m.windowIDValue(),
			"viewport_width":  int(preset.Width),
			"viewport_height": int(preset.Height),
		}, 25*time.Second)
		if err != nil {
			m.setError(err)
			return err
		}
		var out struct {
			Viewport Viewport `json:"viewport"`
			Clamped  bool     `json:"clamped"`
		}
		if err := json.Unmarshal(data, &out); err != nil {
			return fmt.Errorf("decode resize: %w", err)
		}
		if out.Viewport.Width > 0 {
			m.mu.Lock()
			// The window job measured the live page, so this is the achieved
			// viewport and not the requested one.
			m.viewport = Viewport{Width: out.Viewport.Width, Height: out.Viewport.Height, DPR: out.Viewport.DPR}
			m.viewportAt = time.Now()
			m.mu.Unlock()
		}
	}
	m.mu.Lock()
	m.preset = name
	m.presetApplied = true
	mode := m.mode
	sessionID := m.sessionIDValueLocked()
	m.mu.Unlock()
	// Keep the chosen session: a preset click must not make the next launch
	// reopen a different window. A discovered window's synthetic id is
	// ephemeral, so it must never replace the saved normal session.
	saveSettings(mode, name, sessionID)
	if err := m.refreshViewport(ctx); err != nil {
		m.setError(err)
	}
	m.hub.kick()
	return nil
}

func (m *Manager) windowIDValue() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.windowID
}

func (m *Manager) contentTabIDValue() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.contentTabID
}

// SetMode switches the stream between cast and shot and remembers the choice.
func (m *Manager) SetMode(mode string) error {
	if mode != ModeCast && mode != ModeShot {
		return fmt.Errorf("unknown mode %q", mode)
	}
	m.mu.Lock()
	prev := m.mode
	m.mode = mode
	preset := m.preset
	sessionID := m.sessionIDValueLocked()
	m.mu.Unlock()
	if prev != mode {
		saveSettings(mode, preset, sessionID)
	}
	m.hub.kick()
	return nil
}

// refreshViewport re-measures the page's CSS viewport. Input coordinates are
// normalized, so this is only needed to describe the page and to size the
// placeholder; still, a stale value makes taps land off target.
func (m *Manager) refreshViewport(ctx context.Context) error {
	tab := m.contentTab()
	if tab == 0 {
		return errNoPageTab
	}
	data, err := m.job(ctx, tab, "eval", map[string]any{
		"expression": "JSON.stringify({w:window.innerWidth,h:window.innerHeight,dpr:window.devicePixelRatio||1})",
	}, 15*time.Second)
	if err != nil {
		return err
	}
	var out struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return fmt.Errorf("decode eval: %w", err)
	}
	var vp struct {
		W   float64 `json:"w"`
		H   float64 `json:"h"`
		DPR float64 `json:"dpr"`
	}
	if err := json.Unmarshal([]byte(out.Value), &vp); err != nil {
		return fmt.Errorf("decode viewport: %w", err)
	}
	if vp.W > 0 && vp.H > 0 {
		m.mu.Lock()
		m.viewport = Viewport{Width: vp.W, Height: vp.H, DPR: vp.DPR}
		m.viewportAt = time.Now()
		m.mu.Unlock()
	}
	return nil
}

// viewportSize returns the cached CSS viewport, refreshing it when stale.
func (m *Manager) viewportSize(ctx context.Context) (Viewport, error) {
	m.mu.Lock()
	vp := m.viewport
	at := m.viewportAt
	m.mu.Unlock()
	if vp.Width > 0 && time.Since(at) < 10*time.Second {
		return vp, nil
	}
	if err := m.refreshViewport(ctx); err != nil {
		if vp.Width > 0 {
			return vp, nil
		}
		return Viewport{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.viewport, nil
}

// Capture returns one JPEG frame of the driven tab.
func (m *Manager) Capture(ctx context.Context) ([]byte, error) {
	tab := m.contentTab()
	if tab == 0 {
		return nil, errNoPageTab
	}
	data, err := m.job(ctx, tab, "screenshot", map[string]any{
		"format":  "jpeg",
		"quality": captureQuality,
	}, 20*time.Second)
	if err != nil {
		return nil, err
	}
	var out struct {
		Base64 string `json:"base64"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode screenshot: %w", err)
	}
	raw, err := decodeBase64(out.Base64)
	if err != nil {
		return nil, fmt.Errorf("decode screenshot payload: %w", err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty screenshot")
	}
	return raw, nil
}

// SendInput dispatches one pointer, wheel, key, or text event to the page.
func (m *Manager) SendInput(ctx context.Context, ev InputEvent) error {
	tab := m.contentTab()
	if tab == 0 {
		return errNoPageTab
	}
	switch ev.Kind {
	case InputTap, InputDown, InputMove, InputUp, InputWheel:
		vp, err := m.viewportSize(ctx)
		if err != nil {
			return err
		}
		x := ev.X * vp.Width
		y := ev.Y * vp.Height
		params := map[string]any{"x": round2(x), "y": round2(y)}
		switch ev.Kind {
		case InputTap:
			if err := m.dispatchMouse(ctx, tab, "mousePressed", ev.Button, 1, params); err != nil {
				return err
			}
			return m.dispatchMouse(ctx, tab, "mouseReleased", ev.Button, 0, params)
		case InputDown:
			return m.dispatchMouse(ctx, tab, "mousePressed", ev.Button, 1, params)
		case InputUp:
			return m.dispatchMouse(ctx, tab, "mouseReleased", ev.Button, 0, params)
		case InputMove:
			buttons := 0
			if ev.Button != ButtonNone {
				buttons = 1
			}
			return m.dispatchMouse(ctx, tab, "mouseMoved", ev.Button, buttons, params)
		case InputWheel:
			params["type"] = "mouseWheel"
			params["deltaX"] = round2(ev.DeltaX)
			params["deltaY"] = round2(ev.DeltaY)
			_, err := m.job(ctx, tab, "cdp", map[string]any{
				"method": "Input.dispatchMouseEvent",
				"params": params,
			}, 15*time.Second)
			return err
		}
	case InputKey:
		return m.dispatchKey(ctx, tab, ev)
	case InputText:
		if ev.Text == "" {
			return nil
		}
		_, err := m.job(ctx, tab, "cdp", map[string]any{
			"method": "Input.insertText",
			"params": map[string]any{"text": ev.Text},
		}, 15*time.Second)
		return err
	}
	return nil
}

func (m *Manager) dispatchMouse(ctx context.Context, tab int64, typ string, button int, buttons int, params map[string]any) error {
	params["type"] = typ
	params["button"] = buttonName(button)
	params["buttons"] = buttons
	if typ == "mousePressed" {
		params["clickCount"] = 1
	}
	_, err := m.job(ctx, tab, "cdp", map[string]any{
		"method": "Input.dispatchMouseEvent",
		"params": params,
	}, 15*time.Second)
	return err
}

// dispatchKey sends a real key event so the page sees Enter, Tab, or an arrow
// key exactly like it would from the host keyboard.
func (m *Manager) dispatchKey(ctx context.Context, tab int64, ev InputEvent) error {
	key := ev.Key
	if key == "" {
		key = ev.Text
	}
	if key == "" {
		return nil
	}
	vk := ev.VK
	if vk == 0 {
		vk = virtualKey(key)
	}
	code := ev.Code
	if code == "" {
		code = key
	}
	base := map[string]any{
		"key":                   key,
		"code":                  code,
		"windowsVirtualKeyCode": vk,
		"nativeVirtualKeyCode":  vk,
		"modifiers":             ev.Modifiers,
	}
	down := cloneParams(base)
	down["type"] = "keyDown"
	if txt := keyText(key, ev.Modifiers); txt != "" {
		down["text"] = txt
		down["unmodifiedText"] = txt
	}
	if _, err := m.job(ctx, tab, "cdp", map[string]any{
		"method": "Input.dispatchKeyEvent",
		"params": down,
	}, 15*time.Second); err != nil {
		return err
	}
	up := cloneParams(base)
	up["type"] = "keyUp"
	_, err := m.job(ctx, tab, "cdp", map[string]any{
		"method": "Input.dispatchKeyEvent",
		"params": up,
	}, 15*time.Second)
	return err
}

// Status is the JSON body of /api/browser/status.
type Status struct {
	OK       bool          `json:"ok"`
	Daemon   DaemonStatus  `json:"daemon"`
	Session  SessionStatus `json:"session"`
	Capture  CaptureStatus `json:"capture"`
	Preset   string        `json:"preset"`
	Mode     string        `json:"mode"`
	Viewport Viewport      `json:"viewport"`
	Presets  []PresetInfo  `json:"presets"`
	// Tabs is the tab list of the current session, active tab first in Chrome
	// order. The rail renders it for every session; the tab strip only for the
	// current one, which is what ActiveTabID points at.
	Tabs        []TabInfo `json:"tabs"`
	ActiveTabID int64     `json:"active_tab_id,omitempty"`
	// ReadOnly is set when the daemon predates tab multiplexing, so the UI can
	// explain instead of failing each click.
	ReadOnly bool   `json:"read_only,omitempty"`
	Startup  string `json:"startup,omitempty"`
	Error    string `json:"error,omitempty"`
}

// DaemonStatus describes the browser-agent daemon.
type DaemonStatus struct {
	Running bool   `json:"running"`
	Port    int    `json:"port"`
	Version string `json:"version,omitempty"`
}

// SessionStatus describes the attached Chrome window.
type SessionStatus struct {
	ID           string `json:"id"`
	Connected    bool   `json:"connected"`
	WindowID     int64  `json:"window_id,omitempty"`
	ContentTabID int64  `json:"content_tab_id,omitempty"`
	URL          string `json:"url,omitempty"`
	// ExtensionPath is where Chrome must load the unpacked extension from; its
	// id comes from this path, so a copy elsewhere cannot attach.
	ExtensionPath string `json:"extension_path,omitempty"`
}

// CaptureStatus describes the frame stream.
type CaptureStatus struct {
	Active   bool `json:"active"`
	Viewers  int  `json:"viewers"`
	FPS      int  `json:"fps"`
	Frames   int  `json:"frames"`
	LastMS   int  `json:"last_ms"`
	LastSize int  `json:"last_size"`
}

// PresetInfo is one entry of the size switcher.
type PresetInfo struct {
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Snapshot reports the current state without blocking on the browser.
func (m *Manager) Snapshot() Status {
	m.mu.Lock()
	sessionID := m.sessionID
	if sessionID == "" {
		sessionID = SessionID
	}
	st := Status{
		OK:       m.lastErr == "",
		Preset:   m.preset,
		Mode:     m.mode,
		Viewport: m.viewport,
		Daemon:   DaemonStatus{Running: m.daemonVersion != "", Port: m.port, Version: m.daemonVersion},
		Session: SessionStatus{
			ID:            sessionID,
			Connected:     m.connected,
			WindowID:      m.windowID,
			ContentTabID:  m.contentTabID,
			URL:           m.url,
			ExtensionPath: m.extPath,
		},
		Tabs:        cloneTabs(m.tabs),
		ActiveTabID: m.drivenTabLocked(),
		ReadOnly:    m.readOnly,
		Startup:     m.startup,
		Error:       m.lastErr,
	}
	m.mu.Unlock()
	for _, name := range presetOrder {
		vp := presets[name]
		st.Presets = append(st.Presets, PresetInfo{Name: name, Width: int(vp.Width), Height: int(vp.Height)})
	}
	if m.hub != nil {
		st.Capture = m.hub.stats()
	}
	return st
}
