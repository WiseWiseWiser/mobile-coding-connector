package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
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
type Config struct {
	Mode      string `json:"mode"`
	Preset    string `json:"preset"`
	SessionID string `json:"session_id,omitempty"`
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
	job(ctx context.Context, sessionID string, tabID int64, typ string, params any, timeout time.Duration) (json.RawMessage, error)
}

// Manager owns the browser session: it makes sure the daemon, extension, and
// window exist, resizes that window, streams frames, and forwards input.
type Manager struct {
	mu       sync.Mutex
	ensureMu sync.Mutex

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
	lastErr       string
	ensuredAt     time.Time
	lastOpenAt    time.Time
	viewportAt    time.Time
	tabsAt        time.Time
	// newWindowFor marks the session whose page must open in a fresh Chrome
	// window (the + New window path); every other session page reuses the
	// window it already has.
	newWindowFor string
	// readOnly is set when the daemon is too old for tab multiplexing, so the
	// rail can say why instead of firing jobs the extension would reject.
	readOnly bool

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
	return m.d.job(ctx, m.sessionIDValue(), tabID, typ, params, timeout)
}

// jobIn runs a job in an explicit session (the rail reads tabs of sessions the
// user is not currently watching).
func (m *Manager) jobIn(ctx context.Context, sessionID string, tabID int64, typ string, params any, timeout time.Duration) (json.RawMessage, error) {
	if sessionID == "" {
		sessionID = m.sessionIDValue()
	}
	return m.d.job(ctx, sessionID, tabID, typ, params, timeout)
}

// Ensure brings up the whole chain: daemon, session, extension, window, tab.
// It is safe to call repeatedly; the expensive steps are skipped when the
// session is already attached.
func (m *Manager) Ensure(ctx context.Context) error {
	m.ensureMu.Lock()
	defer m.ensureMu.Unlock()
	return m.ensureLocked(ctx)
}

// ensureLocked is Ensure with ensureMu already held; a session switch runs it
// inline so two switches cannot register their windows at the same time.
func (m *Manager) ensureLocked(ctx context.Context) error {
	if err := m.ensureDaemon(ctx); err != nil {
		m.setError(err)
		return err
	}
	if err := m.ensureSession(ctx); err != nil {
		m.setError(err)
		return err
	}
	if err := m.ensureWindow(ctx); err != nil {
		m.setError(err)
		return err
	}
	if err := m.refreshViewport(ctx); err != nil {
		// A page that cannot be measured is still usable; keep the error for the
		// status line but do not fail the whole ensure.
		m.setError(err)
	}
	m.mu.Lock()
	m.ensuredAt = time.Now()
	m.mu.Unlock()
	m.setError(nil)
	return nil
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
	m.mu.Unlock()
}

func (m *Manager) ensureSession(ctx context.Context) error {
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
		m.connected = true
		m.extPath = row.ExtensionInstallPath
		m.mu.Unlock()
		return nil
	}
	// No extension yet: if the session page never loaded, open it once. Chrome
	// reuses the existing tab when the URL matches, so this does not spam tabs.
	m.mu.Lock()
	shouldOpen := row.SessionPageCount == 0 && time.Since(m.lastOpenAt) > 10*time.Second
	if shouldOpen {
		m.lastOpenAt = time.Now()
	}
	m.mu.Unlock()
	if shouldOpen {
		url := m.sessionPageURL(sessionID)
		opener := m.opener
		if m.takeNewWindow(sessionID) {
			opener = m.windowOpener
		}
		if err := opener(ctx, url); err != nil {
			return fmt.Errorf("open the browser-agent session page: %w", err)
		}
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
			// The session page calls chrome.runtime.connect() with the id derived
			// from this directory, so it is the one directory that works.
			if path := row.ExtensionInstallPath; path != "" {
				return fmt.Errorf("load the browser-agent extension in Chrome: Load unpacked from %s", path)
			}
			return fmt.Errorf("load the browser-agent extension in Chrome (1.0.20+), then open the session page")
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
func (m *Manager) ensureWindow(ctx context.Context) error {
	windowID, tabID, url, err := m.resolveTargets(ctx)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.windowID = windowID
	m.contentTabID = tabID
	if url != "" {
		m.url = url
	}
	m.mu.Unlock()

	if !m.macKnown {
		if bounds, err := m.windowBounds(ctx); err == nil {
			m.mu.Lock()
			m.macBounds = bounds
			m.macKnown = true
			m.mu.Unlock()
		}
	}
	return nil
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
	tabID, url = chooseContentTab(info)
	if tabID == 0 {
		created, err := m.createContentTab(ctx, windowID)
		if err != nil {
			return 0, 0, "", err
		}
		tabID = created
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
	return "https://www.google.com/search?q=" + strings.ReplaceAll(s, " ", "+")
}

// SetPreset resizes the window so its viewport is as close to the preset as the
// OS allows, then reports the achieved size. macOS clamps Chrome to a minimum
// width and to the display work area, so this never lies about the result.
func (m *Manager) SetPreset(ctx context.Context, name string) error {
	preset, ok := presets[name]
	if !ok {
		return fmt.Errorf("unknown preset %q", name)
	}
	if err := m.Ensure(ctx); err != nil {
		return err
	}
	// Remember the Mac-sized window before the first shrink so the Mac preset
	// can restore it exactly instead of guessing 1440x900.
	if name != "mac" && !m.macKnown {
		if bounds, err := m.windowBounds(ctx); err == nil && bounds.Width > 0 {
			m.mu.Lock()
			m.macBounds = bounds
			m.macKnown = true
			m.mu.Unlock()
		}
	}
	if name == "mac" && m.macKnown {
		m.mu.Lock()
		bounds := m.macBounds
		windowID := m.windowID
		m.mu.Unlock()
		if _, err := m.job(ctx, m.contentTab(), "window", map[string]any{
			"op":        "update",
			"window_id": windowID,
			"left":      bounds.Left,
			"top":       bounds.Top,
			"width":     bounds.Width,
			"height":    bounds.Height,
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
	mode := m.mode
	m.mu.Unlock()
	_ = saveConfig(Config{Mode: mode, Preset: name})
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

// SetMode switches the stream between cast and shot and remembers the choice.
func (m *Manager) SetMode(mode string) error {
	if mode != ModeCast && mode != ModeShot {
		return fmt.Errorf("unknown mode %q", mode)
	}
	m.mu.Lock()
	prev := m.mode
	m.mode = mode
	preset := m.preset
	m.mu.Unlock()
	if prev != mode {
		_ = saveConfig(Config{Mode: mode, Preset: preset})
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
