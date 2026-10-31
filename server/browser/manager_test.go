package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/server/config"
)

// fakeClient stands in for the browser-agent daemon. The manager's job is to
// translate intent into daemon calls, so the tests assert on the calls it
// makes rather than on Chrome's behaviour.
type fakeClient struct {
	mu      sync.Mutex
	calls   []fakeCall
	created []string
	info    string
	window  string
	// vpW/vpH are the live page viewport. A resize sets them, and eval reads
	// them back, so a test can model a window that macOS clamped.
	vpW, vpH float64
	// rows is what /v1/sessions reports; handle can script a job that depends on
	// earlier calls (a tab that closes, a tab list that changes).
	rows        []sessionRow
	handle      func(fakeCall) (json.RawMessage, bool)
	fail        func(fakeCall) error
	sessionFn   func(string) (*sessionRow, error)
	sessionsErr error
	freshFn     func(string) error
	// direct controls /v1/health direct_supported; default false so existing
	// tests keep the session-only track.
	direct bool
	// windowsPayload answers windowsDirect; nil means the endpoint fails.
	windowsPayload string
	// directCalls records jobDirect invocations in order.
	directCalls []fakeCall
	// directHandle answers a direct job; fallthrough returns {}.
	directHandle func(fakeCall) (json.RawMessage, bool)
	// directFail makes jobDirect (and windowsDirect) fail when set.
	directFail error
	// directErrFor scripts per-call direct failures (window-gone etc.).
	directErrFor func(fakeCall) error
}

type fakeCall struct {
	Session string
	Tab     int64
	Type    string
	Params  map[string]any
	// Direct marks a jobDirect invocation (window-addressed, no session).
	Direct bool
}

func (f *fakeClient) health(ctx context.Context) (*daemonHealth, error) {
	f.mu.Lock()
	direct := f.direct
	f.mu.Unlock()
	return &daemonHealth{OK: true, Version: "1.0.23", DirectSupported: direct}, nil
}

func (f *fakeClient) jobDirect(ctx context.Context, windowID, tabID int64, typ string, params any, timeout time.Duration) (json.RawMessage, error) {
	call := fakeCall{Session: sessionlessSessionID(windowID), Tab: tabID, Type: typ, Direct: true}
	if p, ok := params.(map[string]any); ok {
		call.Params = make(map[string]any, len(p))
		for k, v := range p {
			call.Params[k] = v
		}
	}
	f.mu.Lock()
	f.directCalls = append(f.directCalls, call)
	handle, fail, errFor := f.directHandle, f.directFail, f.directErrFor
	f.mu.Unlock()
	if fail != nil {
		return nil, fail
	}
	if errFor != nil {
		if err := errFor(call); err != nil {
			return nil, err
		}
	}
	if handle != nil {
		if data, ok := handle(call); ok {
			return data, nil
		}
	}
	switch typ {
	case "info":
		f.mu.Lock()
		info := f.info
		f.mu.Unlock()
		return json.RawMessage(info), nil
	case "eval":
		f.mu.Lock()
		vpW, vpH := f.vpW, f.vpH
		f.mu.Unlock()
		return json.RawMessage(fmt.Sprintf(`{"value":"{\"w\":%g,\"h\":%g,\"dpr\":2}"}`, vpW, vpH)), nil
	case "create_tab":
		return json.RawMessage(`{"tab_id":901,"window_id":901}`), nil
	}
	return json.RawMessage(`{}`), nil
}

func (f *fakeClient) windowsDirect(ctx context.Context) (*windowsPayload, error) {
	f.mu.Lock()
	raw, fail := f.windowsPayload, f.directFail
	f.mu.Unlock()
	if fail != nil {
		return nil, fail
	}
	if raw == "" {
		return nil, fmt.Errorf("direct windows unavailable")
	}
	var payload windowsPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func (f *fakeClient) sessions(ctx context.Context) ([]sessionRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rows, f.sessionsErr
}

func (f *fakeClient) session(ctx context.Context, id string) (*sessionRow, error) {
	if f.sessionFn != nil {
		return f.sessionFn(id)
	}
	return &sessionRow{SessionID: id, Extension: extensionInfo{Connected: true}, SessionPageCount: 1}, nil
}

func (f *fakeClient) createSession(ctx context.Context, id string) error {
	f.mu.Lock()
	f.created = append(f.created, id)
	f.mu.Unlock()
	return nil
}

func (f *fakeClient) createFreshSession(ctx context.Context, id string) error {
	if f.freshFn != nil {
		if err := f.freshFn(id); err != nil {
			return err
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, row := range f.rows {
		if row.SessionID == id {
			return errSessionAlreadyExists
		}
	}
	f.created = append(f.created, id)
	f.rows = append(f.rows, sessionRow{SessionID: id})
	return nil
}

func (f *fakeClient) job(ctx context.Context, session string, tab int64, typ string, params any, timeout time.Duration) (json.RawMessage, error) {
	call := fakeCall{Session: session, Tab: tab, Type: typ}
	if p, ok := params.(map[string]any); ok {
		// Snapshot: dispatchMouse mutates the same map for press and release,
		// so keeping a pointer would make every recorded call look identical.
		call.Params = make(map[string]any, len(p))
		for k, v := range p {
			if inner, ok := v.(map[string]any); ok {
				cp := make(map[string]any, len(inner))
				for ik, iv := range inner {
					cp[ik] = iv
				}
				v = cp
			}
			call.Params[k] = v
		}
	}
	f.mu.Lock()
	f.calls = append(f.calls, call)
	handle := f.handle
	f.mu.Unlock()
	if f.fail != nil {
		if err := f.fail(call); err != nil {
			return nil, err
		}
	}
	if handle != nil {
		if data, ok := handle(call); ok {
			return data, nil
		}
	}
	switch typ {
	case "info":
		return json.RawMessage(f.info), nil
	case "window":
		if op, _ := call.Params["op"].(string); op == "resize" {
			// The achieved viewport is what the window really has after macOS
			// clamps, which is narrower than the nominal preset.
			f.mu.Lock()
			f.vpW, f.vpH = 500, 844
			f.mu.Unlock()
			return json.RawMessage(`{"viewport":{"width":500,"height":844,"dpr":2},"clamped":true}`), nil
		}
		return json.RawMessage(f.window), nil
	case "eval":
		f.mu.Lock()
		vpW, vpH := f.vpW, f.vpH
		f.mu.Unlock()
		return json.RawMessage(fmt.Sprintf(`{"value":"{\"w\":%g,\"h\":%g,\"dpr\":2}"}`, vpW, vpH)), nil
	}
	return json.RawMessage(`{}`), nil
}

func (f *fakeClient) last(typ string) (fakeCall, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := len(f.calls) - 1; i >= 0; i-- {
		if f.calls[i].Type == typ {
			return f.calls[i], true
		}
	}
	return fakeCall{}, false
}

func (f *fakeClient) all(typ string) []fakeCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []fakeCall
	for _, c := range f.calls {
		if c.Type == typ {
			out = append(out, c)
		}
	}
	return out
}

func newTestManager(t *testing.T, f *fakeClient) *Manager {
	t.Helper()
	// Preset and mode are remembered in .ai-critic/browser-config.json; keep
	// tests from writing into the checkout (or over a real user's choice).
	prevDataDir := config.DataDir
	config.DataDir = t.TempDir()
	t.Cleanup(func() { config.DataDir = prevDataDir })

	m := &Manager{d: f, port: 43799, preset: "mac", mode: ModeCast}
	m.hub = newHub(m)
	return m
}

// healthyFake scripts the responses a first Ensure needs.
func healthyFake() *fakeClient {
	return &fakeClient{
		info: `{"window_id":900,"tabs":[` +
			`{"id":901,"url":"http://127.0.0.1:43799/go?session=ai-critic","active":true},` +
			`{"id":902,"url":"data:text/html;charset=utf-8,%3Ctitle%3Eplaceholder%3C/title%3E","active":false}]}`,
		window: `{"window_id":900,"window":{"left":10,"top":20,"width":1200,"height":1040}}`,
		vpW:    1200,
		vpH:    901,
	}
}

func TestResolveTargetsSkipsSessionPage(t *testing.T) {
	f := healthyFake()
	m := newTestManager(t, f)
	windowID, tabID, _, err := m.resolveTargets(context.Background())
	if err != nil {
		t.Fatalf("resolveTargets: %v", err)
	}
	if windowID != 900 {
		t.Fatalf("window = %d, want 900", windowID)
	}
	if tabID != 902 {
		t.Fatalf("tab = %d, want the content tab 902 (never the /go session page 901)", tabID)
	}
}

// A preset asks for a nominal size, but macOS clamps windows that do not fit
// the display. Status must carry what Chrome actually has, not what we asked
// for, or the phone draws the cursor offset from the real page.
// A transient navigation failure must not outlive the recovery it describes:
// the stream records what it surfaced and retires exactly that error on the
// next success, leaving errors from other paths (preset apply, window ops)
// untouched.
func TestHubRecordSuccessRetiresOnlyItsOwnError(t *testing.T) {
	f := healthyFake()
	m := newTestManager(t, f)
	h := &hub{m: m, viewers: make(map[*viewer]struct{}), kickCh: make(chan struct{}, 1), inputCh: make(chan InputEvent, 1)}

	navErr := errors.New("Inspected target navigated or closed (url changed)")
	h.recordFailure(navErr)
	if snap := m.Snapshot(); snap.Error != navErr.Error() {
		t.Fatalf("status error = %q, want the surfaced navigation error", snap.Error)
	}

	// An unrelated, later error must survive a stream success.
	h.recordFailure(navErr)
	m.setError(errors.New("unknown preset \"mac\""))
	h.recordSuccess()
	if snap := m.Snapshot(); snap.Error != `unknown preset "mac"` {
		t.Fatalf("status error = %q, want the newer unrelated error kept", snap.Error)
	}

	// The stream's own error is retired by the next success.
	h.recordFailure(navErr)
	h.recordSuccess()
	if snap := m.Snapshot(); snap.Error != "" {
		t.Fatalf("status error = %q, want it cleared after recovery", snap.Error)
	}
	// A second success with nothing recorded is a no-op.
	h.recordSuccess()
	if snap := m.Snapshot(); snap.Error != "" {
		t.Fatalf("status error = %q after idle success, want empty", snap.Error)
	}
}

func TestSetPresetReportsAchievedViewport(t *testing.T) {
	f := healthyFake()
	m := newTestManager(t, f)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.SetPreset(ctx, "iphone"); err != nil {
		t.Fatalf("SetPreset: %v", err)
	}
	call, ok := f.last("window")
	if !ok || call.Params["op"] != "resize" {
		t.Fatalf("no resize job, calls: %+v", f.calls)
	}
	if call.Params["viewport_width"] != 390 || call.Params["viewport_height"] != 844 {
		t.Fatalf("resize params = %v, want the nominal 390x844", call.Params)
	}
	snap := m.Snapshot()
	if snap.Viewport.Width != 500 || snap.Viewport.Height != 844 {
		t.Fatalf("reported viewport = %+v, want the achieved 500x844", snap.Viewport)
	}
	if snap.Preset != "iphone" {
		t.Fatalf("preset = %q, want iphone", snap.Preset)
	}
}

func TestMacPresetRestoresCapturedBounds(t *testing.T) {
	f := healthyFake()
	m := newTestManager(t, f)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.Ensure(ctx); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if err := m.SetPreset(ctx, "ipad"); err != nil {
		t.Fatalf("SetPreset(ipad): %v", err)
	}
	if err := m.SetPreset(ctx, "mac"); err != nil {
		t.Fatalf("SetPreset(mac): %v", err)
	}
	call, ok := f.last("window")
	if !ok || call.Params["op"] != "update" {
		t.Fatalf("mac preset did not restore bounds, calls: %+v", f.calls)
	}
	if call.Params["width"] != 1200 || call.Params["height"] != 1040 {
		t.Fatalf("restored bounds = %v, want the captured 1200x1040", call.Params)
	}
	if call.Params["state"] != "normal" {
		t.Fatalf("state = %v, want normal before resizing", call.Params["state"])
	}
}

// The remembered Mac frame survives restarts: a manager whose memory was
// seeded from browser-config.json must restore via the bounds on disk (the
// exact op:update path) instead of guessing from the page-viewport
// measurement, which a concurrently-debugged tab can poison.
func TestMacPresetRestoresBoundsPersistedAcrossRestart(t *testing.T) {
	saved := Bounds{Left: 40, Top: 30, Width: 1200, Height: 1040}
	f := healthyFake()
	// The window is already phone-sized after the restart, so nothing can be
	// measured fresh — only the persisted frame can restore it.
	f.window = `{"window_id":900,"window":{"left":10,"top":20,"width":500,"height":900}}`
	f.vpW, f.vpH = 500, 844
	m := newTestManager(t, f)
	// newTestManager points config.DataDir at a temp dir; the persisted frame
	// is written there, the way the previous process would have left it.
	if err := saveConfig(Config{Mode: ModeShot, Preset: "iphone", SessionID: SessionID, MacBounds: &saved}); err != nil {
		t.Fatalf("saveConfig: %v", err)
	}
	m.preset = "iphone"
	// Model the post-restart state directly: memory seeded from disk.
	m.macBounds = saved
	m.macKnown = true
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.SetPreset(ctx, "mac"); err != nil {
		t.Fatalf("SetPreset(mac): %v", err)
	}
	call, ok := f.last("window")
	if !ok || call.Params["op"] != "update" {
		t.Fatalf("mac preset did not use the persisted bounds, calls: %+v", f.calls)
	}
	for k, want := range map[string]int{"left": 40, "top": 30, "width": 1200, "height": 1040} {
		if got, _ := call.Params[k].(int); got != want {
			t.Fatalf("update %s = %v, want %d", k, call.Params[k], want)
		}
	}
	// The restored frame is re-recorded so later resizes keep it fresh on disk.
	stored := loadConfig()
	if stored.MacBounds == nil || *stored.MacBounds != saved {
		t.Fatalf("persisted mac bounds = %+v, want %+v", stored.MacBounds, saved)
	}
}

// A saved preset is applied to a window that has not received it yet, so a
// restart cannot leave the switcher saying "iPhone" over a desktop-sized page.
func TestEnsureAppliesSavedPresetToNewWindow(t *testing.T) {
	f := healthyFake()
	m := newTestManager(t, f)
	m.preset = "iphone"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.Ensure(ctx); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	call, ok := f.last("window")
	if !ok || call.Params["op"] != "resize" {
		t.Fatalf("saved preset was not applied, calls: %+v", f.calls)
	}
	if call.Params["viewport_width"] != 390 || call.Params["viewport_height"] != 844 {
		t.Fatalf("resize params = %v, want the saved 390x844", call.Params)
	}
}

// Preset and mode are saved separately from the session id, but they live in
// one config file. Saving only the setting must not drop the window the person
// switched to.
func TestPresetAndModeKeepSelectedSession(t *testing.T) {
	f := healthyFake()
	m := newTestManager(t, f)
	m.sessionID = "sess-other"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.SetPreset(ctx, "iphone"); err != nil {
		t.Fatalf("SetPreset: %v", err)
	}
	if got := loadConfig().SessionID; got != "sess-other" {
		t.Fatalf("session after SetPreset = %q, want sess-other", got)
	}
	if err := m.SetMode(ModeShot); err != nil {
		t.Fatalf("SetMode: %v", err)
	}
	if got := loadConfig().SessionID; got != "sess-other" {
		t.Fatalf("session after SetMode = %q, want sess-other", got)
	}
}

// A restart can find the window still at phone size while the saved preset is
// Mac. That size must not be captured as the Mac bounds, or selecting Mac later
// would faithfully restore 500x844.
func TestMacPresetIgnoresPhoneSizedWindow(t *testing.T) {
	f := healthyFake()
	f.window = `{"window_id":900,"window":{"left":10,"top":20,"width":500,"height":900}}`
	f.vpW, f.vpH = 500, 844
	m := newTestManager(t, f)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.Ensure(ctx); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	call, ok := f.last("window")
	if !ok || call.Params["op"] != "resize" {
		t.Fatalf("Mac preset did not ask for a desktop size, calls: %+v", f.calls)
	}
	if call.Params["viewport_width"] != 1440 || call.Params["viewport_height"] != 900 {
		t.Fatalf("resize params = %v, want the nominal 1440x900", call.Params)
	}
	m.mu.Lock()
	known, bounds := m.macKnown, m.macBounds
	m.mu.Unlock()
	if known {
		t.Fatalf("phone-sized window captured as Mac bounds: %+v", bounds)
	}
}

// iPad is only 834 wide, below the desktop threshold. A window left at iPad
// size must not become the target the Mac preset restores either.
func TestMacPresetIgnoresTabletSizedWindow(t *testing.T) {
	f := healthyFake()
	f.window = `{"window_id":900,"window":{"left":10,"top":20,"width":834,"height":1194}}`
	f.vpW, f.vpH = 834, 1194
	m := newTestManager(t, f)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.Ensure(ctx); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	call, ok := f.last("window")
	if !ok || call.Params["op"] != "resize" {
		t.Fatalf("Mac preset did not ask for a desktop size, calls: %+v", f.calls)
	}
	if call.Params["viewport_width"] != 1440 || call.Params["viewport_height"] != 900 {
		t.Fatalf("resize params = %v, want the nominal 1440x900", call.Params)
	}
	m.mu.Lock()
	known, bounds := m.macKnown, m.macBounds
	m.mu.Unlock()
	if known {
		t.Fatalf("iPad-sized window captured as Mac bounds: %+v", bounds)
	}
}

// The client sends normalized coordinates so it never has to track the
// window's real size; the server is the only place that knows the viewport.
func TestSendInputMapsNormalizedCoordinates(t *testing.T) {
	f := healthyFake()
	m := newTestManager(t, f)
	m.mu.Lock()
	m.contentTabID = 902
	m.viewport = Viewport{Width: 500, Height: 844, DPR: 2}
	m.viewportAt = time.Now()
	m.mu.Unlock()

	if err := m.SendInput(context.Background(), InputEvent{
		Kind: InputTap, X: 0.5, Y: 0.25, Button: ButtonLeft,
	}); err != nil {
		t.Fatalf("SendInput: %v", err)
	}
	calls := f.all("cdp")
	if len(calls) != 2 {
		t.Fatalf("cdp calls = %d, want press + release", len(calls))
	}
	for i, want := range []string{"mousePressed", "mouseReleased"} {
		params, _ := calls[i].Params["params"].(map[string]any)
		if params["type"] != want {
			t.Fatalf("call %d type = %v, want %s", i, params["type"], want)
		}
		if params["x"] != 250.0 || params["y"] != 211.0 {
			t.Fatalf("call %d point = (%v,%v), want (250,211)", i, params["x"], params["y"])
		}
		if params["button"] != "left" {
			t.Fatalf("call %d button = %v, want left", i, params["button"])
		}
		if calls[i].Tab != 902 {
			t.Fatalf("call %d tab = %d, want the content tab", i, calls[i].Tab)
		}
	}
}

// Before Ensure has opened a content tab there is nothing to drive: callers
// need a sentinel they can treat as "still warming up" instead of an error.
func TestSendInputWithoutContentTab(t *testing.T) {
	m := newTestManager(t, healthyFake())
	err := m.SendInput(context.Background(), InputEvent{Kind: InputTap, X: 1, Y: 1, Button: ButtonLeft})
	if err != errNoPageTab {
		t.Fatalf("err = %v, want errNoPageTab", err)
	}
	if _, err := m.Capture(context.Background()); err != errNoPageTab {
		t.Fatalf("Capture err = %v, want errNoPageTab", err)
	}
}

// /v1/session only accepts ?session=; ?session_id= is a 400. Getting this
// wrong parks the whole page on "connecting extension" forever.
func TestDaemonSessionUsesSessionQuery(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/session" {
			t.Errorf("path = %s, want /v1/session", r.URL.Path)
		}
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"session_id":"ai-critic","extension":{"connected":true}}`))
	}))
	defer srv.Close()

	d := &daemon{base: srv.URL, client: srv.Client()}
	row, err := d.session(context.Background(), SessionID)
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	if gotQuery != "session=ai-critic" {
		t.Fatalf("query = %q, want session=ai-critic", gotQuery)
	}
	if !row.Extension.Connected {
		t.Fatalf("row = %+v, want a connected extension", row)
	}
}

// The viewer only offers http(s). A stale bundle must not be able to drive the
// page to a local file or a browser-internal URL, and a rejected navigation
// must not warm the daemon or open a tab.
func TestNavigateRejectsNonWebScheme(t *testing.T) {
	f := healthyFake()
	m := newTestManager(t, f)
	for _, raw := range []string{"file:///tmp/x", "chrome://version"} {
		if err := m.Navigate(context.Background(), raw); err == nil {
			t.Fatalf("Navigate(%q) succeeded, want a scheme error", raw)
		}
	}
	if len(f.calls) != 0 {
		t.Fatalf("daemon calls = %d, want 0 before validation", len(f.calls))
	}
}

func TestIsWebURL(t *testing.T) {
	for raw, want := range map[string]bool{
		"http://example.com":  true,
		"HTTPS://example.com": true,
		"file:///tmp/x":       false,
		"chrome://version":    false,
		"data:text/html,x":    false,
		"":                    false,
	} {
		if got := isWebURL(raw); got != want {
			t.Errorf("isWebURL(%q) = %v, want %v", raw, got, want)
		}
	}
}
