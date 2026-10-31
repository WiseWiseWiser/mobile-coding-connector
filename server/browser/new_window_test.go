package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func newWindowTestManager(t *testing.T) (*Manager, *fakeClient, *sync.Mutex, map[string]bool) {
	t.Helper()
	f := healthyFake()
	f.rows = []sessionRow{{SessionID: "ai-critic-2", Extension: extensionInfo{Connected: true}, SessionPageCount: 1}}
	m := newTestManager(t, f)
	m.sessionID = "ai-critic-2"
	m.windowID = 44
	m.connected = true
	if err := saveConfig(Config{SessionID: "ai-critic-2"}); err != nil {
		t.Fatal(err)
	}
	mu := &sync.Mutex{}
	connected := map[string]bool{"ai-critic-2": true}
	f.sessionFn = func(id string) (*sessionRow, error) {
		mu.Lock()
		defer mu.Unlock()
		return &sessionRow{SessionID: id, Extension: extensionInfo{Connected: connected[id]}}, nil
	}
	m.windowOpener = func(_ context.Context, raw string) error {
		u, err := url.Parse(raw)
		if err != nil {
			return err
		}
		mu.Lock()
		connected[u.Query().Get("session")] = true
		mu.Unlock()
		return nil
	}
	return m, f, mu, connected
}

func TestDaemonRejectsBadSessionListAndFreshConflicts(t *testing.T) {
	status, body := http.StatusInternalServerError, `{"error":"registry failed"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status); _, _ = w.Write([]byte(body)) }))
	defer srv.Close()
	d := &daemon{base: srv.URL, client: srv.Client()}
	if _, err := d.sessions(context.Background()); err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("HTTP failure accepted: %v", err)
	}
	status, body = http.StatusOK, `{"error":"missing sessions"}`
	if _, err := d.sessions(context.Background()); err == nil {
		t.Fatal("malformed list accepted")
	}
	status, body = http.StatusOK, `{"sessions":[]}`
	if rows, err := d.sessions(context.Background()); err != nil || len(rows) != 0 {
		t.Fatalf("wrapped empty list: %v %v", rows, err)
	}
	status, body = http.StatusConflict, `{"error":"session already exists"}`
	if err := d.createFreshSession(context.Background(), "ai-critic-3"); !errors.Is(err, errSessionAlreadyExists) {
		t.Fatalf("conflict accepted: %v", err)
	}
	if err := d.createSession(context.Background(), "ai-critic-3"); err != nil {
		t.Fatalf("ordinary ensure must remain idempotent: %v", err)
	}
}

func TestNewWindowCreatesUnusedSessionAndOpensWindow(t *testing.T) {
	m, _, _, _ := newWindowTestManager(t)
	id, err := m.NewSession(context.Background())
	if err != nil || id != "ai-critic-3" {
		t.Fatalf("new session = %q, %v", id, err)
	}
	if m.windowID != 900 || m.sessionID != id {
		t.Fatalf("wrong new window: session=%s window=%d", m.sessionID, m.windowID)
	}
	if cfg := loadConfig(); cfg.SessionID != id {
		t.Fatalf("saved session=%s", cfg.SessionID)
	}
}

func TestNewWindowFailurePreservesPreviousSelection(t *testing.T) {
	m, _, _, _ := newWindowTestManager(t)
	m.windowOpener = func(context.Context, string) error { return errors.New("launch failed") }
	_, err := m.NewSession(context.Background())
	if err == nil || !strings.Contains(err.Error(), "launch failed") {
		t.Fatalf("expected error, got %v", err)
	}
	if m.sessionID != "ai-critic-2" || m.windowID != 44 || !m.connected {
		t.Fatalf("previous session not restored: %s %d %v", m.sessionID, m.windowID, m.connected)
	}
	if cfg := loadConfig(); cfg.SessionID != "ai-critic-2" {
		t.Fatalf("saved session changed to %s", cfg.SessionID)
	}
}

func TestNewWindowRejectsExistingWindowBeforeResize(t *testing.T) {
	m, f, _, _ := newWindowTestManager(t)
	f.info = `{"window_id":44,"tabs":[{"id":901,"url":"http://127.0.0.1:43799/go?session=ai-critic-3"}]}`
	_, err := m.NewSession(context.Background())
	if err == nil || !strings.Contains(err.Error(), "existing window") {
		t.Fatalf("expected distinct-window failure, got %v", err)
	}
	if _, ok := f.last("window"); ok {
		t.Fatal("existing window was resized")
	}
	if m.sessionID != "ai-critic-2" || m.windowID != 44 {
		t.Fatal("previous session not restored")
	}
}

func TestNewWindowRejectsOtherExistingWindow(t *testing.T) {
	m, f, _, _ := newWindowTestManager(t)
	f.info = `{"window_id":55,"tabs":[{"id":901,"url":"data:text/html,old"}]}`
	f.handle = func(c fakeCall) (json.RawMessage, bool) {
		if c.Type == "windows" {
			return json.RawMessage(`{"features":["all-windows"],"windows":[{"window_id":44},{"window_id":55}]}`), true
		}
		return nil, false
	}
	_, err := m.NewSession(context.Background())
	if err == nil || !strings.Contains(err.Error(), "existing window") {
		t.Fatalf("expected background window protection, got %v", err)
	}
	if _, ok := f.last("window"); ok {
		t.Fatal("existing background window was resized")
	}
}

func TestNewWindowRegistryFailureDoesNotChangeSelection(t *testing.T) {
	m, f, _, _ := newWindowTestManager(t)
	f.sessionsErr = errors.New("registry unavailable")
	_, err := m.NewSession(context.Background())
	if err == nil || !strings.Contains(err.Error(), "registry unavailable") {
		t.Fatalf("error = %v", err)
	}
	if m.sessionID != "ai-critic-2" || m.windowID != 44 {
		t.Fatal("selection changed after registry error")
	}
	if len(f.created) != 0 {
		t.Fatalf("unexpected sessions created: %v", f.created)
	}
}

// A direct-capable daemon mints no browser-agent session for "+ New window":
// the window opens with a capturable data: page — never the /go control page —
// and is adopted over the direct rail as sess-window-<id>.
func TestNewSessionDirectSkipsSessionAndGoPage(t *testing.T) {
	f := directFake()
	m := newTestManager(t, f)
	m.recordDaemon(&daemonHealth{OK: true, Version: "1.0.27", DirectSupported: true})

	f.mu.Lock()
	base := f.windowsPayload
	f.mu.Unlock()
	var openerURLs []string
	m.windowOpener = func(_ context.Context, url string) error {
		f.mu.Lock()
		openerURLs = append(openerURLs, url)
		// Chrome "opens" a new window carrying only the opener URL. The base
		// payload ends "…}]}"; drop the closing "]}" and splice window 13 in.
		opened := `{"window_id":13,"type":"normal","focused":true,"width":1440,"height":1077,"state":"normal",` +
			`"tabs":[{"tab_id":131,"index":0,"title":"ai-critic","url":"` + directNewWindowURL + `","active":true,"capturable":true,"role":"user"}]}`
		f.windowsPayload = base[:len(base)-2] + "," + opened + "]}"
		f.mu.Unlock()
		return nil
	}
	f.directErrFor = nil // the fixture refuses info for windows other than 11/12
	f.directHandle = func(call fakeCall) (json.RawMessage, bool) {
		switch call.Type {
		case "info":
			if call.Session == sessionlessSessionID(13) {
				return json.RawMessage(`{"window_id":13,"direct":true,"tabs":[` +
					`{"id":131,"index":1,"title":"ai-critic","url":"` + directNewWindowURL + `","active":true,"role":"user"}],` +
					`"job_target":{"tab_id":131,"tab_index":1,"reason":"active_in_window"}}`), true
			}
		case "window", "resize":
			return json.RawMessage(`{}`), true
		}
		return nil, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id, err := m.NewSession(ctx)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if id != sessionlessSessionID(13) {
		t.Fatalf("session id = %q, want the adopted window id", id)
	}
	if len(openerURLs) != 1 {
		t.Fatalf("opener calls = %v, want exactly one", openerURLs)
	}
	if strings.Contains(openerURLs[0], "/go?session=") || !strings.HasPrefix(openerURLs[0], "data:text/html") {
		t.Fatalf("opener url = %q, want the capturable data page without /go", openerURLs[0])
	}
	if len(f.created) != 0 {
		t.Fatalf("direct new window minted browser-agent sessions: %v", f.created)
	}
	if got := m.windowIDValue(); got != 13 {
		t.Fatalf("window = %d, want 13", got)
	}
	if snap := m.Snapshot(); snap.Session.ID != sessionlessSessionID(13) {
		t.Fatalf("snapshot session = %q, want %q", snap.Session.ID, sessionlessSessionID(13))
	}
}

// When the new window never shows up the adoption fails and the previous
// session stays selected.
func TestNewSessionDirectRestoresStateWhenWindowMissing(t *testing.T) {
	f := directFake()
	m := newTestManager(t, f)
	m.recordDaemon(&daemonHealth{OK: true, Version: "1.0.27", DirectSupported: true})
	m.mu.Lock()
	m.sessionID = sessionlessSessionID(11)
	m.windowID = 11
	m.connected = true
	m.mu.Unlock()

	m.windowOpener = func(_ context.Context, _ string) error { return nil } // Chrome opens nothing

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := m.NewSession(ctx); err == nil {
		t.Fatal("NewSession succeeded although no window appeared")
	}
	if got := m.sessionIDValue(); got != sessionlessSessionID(11) || m.windowIDValue() != 11 || !m.connected {
		t.Fatalf("previous selection not restored: %s %d %v", m.sessionIDValue(), m.windowIDValue(), m.connected)
	}
}

// The windows enumeration adopts a fresh direct window as soon as its opener
// tab is capturable, but the info job can still see that tab as about:blank
// for a moment (the data: URL has not committed). Resolve must re-read instead
// of spawning a placeholder tab NEXT TO the opener.
func TestNewSessionDirectWaitsForOpenerBeforePlaceholder(t *testing.T) {
	f := directFake()
	m := newTestManager(t, f)
	m.recordDaemon(&daemonHealth{OK: true, Version: "1.0.27", DirectSupported: true})

	f.mu.Lock()
	base := f.windowsPayload
	f.mu.Unlock()
	m.windowOpener = func(_ context.Context, url string) error {
		f.mu.Lock()
		opened := `{"window_id":13,"type":"normal","focused":true,"width":1440,"height":1077,"state":"normal",` +
			`"tabs":[{"tab_id":131,"index":0,"title":"ai-critic","url":"` + directNewWindowURL + `","active":true,"capturable":true,"role":"user"}]}`
		f.windowsPayload = base[:len(base)-2] + "," + opened + "]}"
		f.mu.Unlock()
		return nil
	}
	f.directErrFor = nil
	infoCalls := 0
	f.directHandle = func(call fakeCall) (json.RawMessage, bool) {
		switch call.Type {
		case "info":
			if call.Session == sessionlessSessionID(13) {
				infoCalls++
				if infoCalls == 1 {
					// The opener tab is still committing: no usable content tab yet.
					return json.RawMessage(`{"window_id":13,"direct":true,"tabs":[]}`), true
				}
				return json.RawMessage(`{"window_id":13,"direct":true,"tabs":[` +
					`{"id":131,"index":1,"title":"ai-critic","url":"` + directNewWindowURL + `","active":true,"role":"user"}],` +
					`"job_target":{"tab_id":131,"tab_index":1,"reason":"active_in_window"}}`), true
			}
		case "window", "resize":
			return json.RawMessage(`{}`), true
		}
		return nil, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id, err := m.NewSession(ctx)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if id != sessionlessSessionID(13) {
		t.Fatalf("session id = %q", id)
	}
	for _, call := range f.directCalls {
		if call.Type == "create_tab" {
			t.Fatal("placeholder tab created although the opener tab committed")
		}
	}
	if m.contentTabIDValue() != 131 {
		t.Fatalf("content tab = %d, want the opener 131", m.contentTabIDValue())
	}
}

func TestNewWindowRetriesReservationConflict(t *testing.T) {
	m, f, _, _ := newWindowTestManager(t)
	f.freshFn = func(id string) error {
		if id == "ai-critic-3" {
			return errSessionAlreadyExists
		}
		return nil
	}
	id, err := m.NewSession(context.Background())
	if err != nil || id != "ai-critic-4" {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func TestNewWindowRejectsAlreadyConnectedReservation(t *testing.T) {
	m, f, _, _ := newWindowTestManager(t)
	f.sessionFn = func(id string) (*sessionRow, error) {
		return &sessionRow{SessionID: id, Extension: extensionInfo{Connected: true}}, nil
	}
	called := false
	m.windowOpener = func(context.Context, string) error { called = true; return nil }
	_, err := m.NewSession(context.Background())
	if err == nil || !strings.Contains(err.Error(), "already connected") || called {
		t.Fatalf("err=%v opener called=%v", err, called)
	}
	if m.sessionID != "ai-critic-2" {
		t.Fatalf("session changed to %s", m.sessionID)
	}
}

func TestConcurrentNewWindowsUseDistinctSessions(t *testing.T) {
	m, f, _, _ := newWindowTestManager(t)
	// Each simulated attachment reports a distinct Chrome window.
	f.handle = func(c fakeCall) (json.RawMessage, bool) {
		if c.Type != "info" {
			return nil, false
		}
		id := 900
		if c.Session == "ai-critic-4" {
			id = 901
		}
		return json.RawMessage(fmt.Sprintf(`{"window_id":%d,"tabs":[{"id":902,"url":"data:text/html,hello"}]}`, id)), true
	}
	var wg sync.WaitGroup
	results := make(chan string, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); id, err := m.NewSession(context.Background()); results <- id; errs <- err }()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	seen := map[string]bool{}
	for id := range results {
		seen[id] = true
	}
	if !seen["ai-critic-3"] || !seen["ai-critic-4"] {
		t.Fatalf("sessions = %v", seen)
	}
}
