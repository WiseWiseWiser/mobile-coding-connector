package browser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNavigateTabTargetsExplicitSessionAndTab(t *testing.T) {
	f := healthyFake()
	f.info = `{"window_id":900,"tabs":[{"id":901,"url":"http://127.0.0.1:43799/go?session=other","role":"session_page"},{"id":904,"url":"https://old.example/","role":"user","active":true}]}`
	m := newTestManager(t, f)
	m.sessionID, m.contentTabID, m.url = "other", 902, "https://current.example/"
	if err := m.NavigateTab(context.Background(), "other", 904, "https://target.example/a?q=1#frag"); err != nil {
		t.Fatal(err)
	}
	call, ok := f.last("cdp")
	if !ok || call.Session != "other" || call.Tab != 904 || call.Params["method"] != "Page.navigate" {
		t.Fatalf("wrong target: %+v", call)
	}
	params := call.Params["params"].(map[string]any)
	if params["url"] != "https://target.example/a?q=1#frag" {
		t.Fatalf("url=%v", params["url"])
	}
	if m.sessionID != "other" || m.contentTabID != 902 || m.url != "https://current.example/" {
		t.Fatal("navigation changed a different current session")
	}
}

func TestNormalizeSearchEscapesSpecialCharacters(t *testing.T) {
	got := normalizeURL("cats & dogs #1")
	if got != "https://www.google.com/search?q=cats+%26+dogs+%231" {
		t.Fatalf("escaped search=%q", got)
	}
}

func TestNavigateTabRefusesReadOnlyDaemon(t *testing.T) {
	f := healthyFake()
	m := newTestManager(t, f)
	m.sessionID = "ai-critic"
	m.readOnly = true
	if err := m.NavigateTab(context.Background(), "ai-critic", 902, "https://example.com"); err == nil {
		t.Fatal("accepted read-only daemon")
	}
	if _, ok := f.last("cdp"); ok {
		t.Fatal("read-only daemon received navigation")
	}
}

func TestNavigateTabRejectsStaleSession(t *testing.T) {
	f := healthyFake()
	m := newTestManager(t, f)
	m.sessionID = "new-current"
	if err := m.NavigateTab(context.Background(), "old-session", 902, "https://example.com"); err == nil {
		t.Fatal("accepted stale session")
	}
	if _, ok := f.last("cdp"); ok {
		t.Fatal("stale session was navigated")
	}
}

func TestNavigateTabRejectsMissingOrInfrastructureTab(t *testing.T) {
	f := healthyFake()
	m := newTestManager(t, f)
	for _, id := range []int64{0, 901, 999} {
		if err := m.NavigateTab(context.Background(), "ai-critic", id, "https://example.com"); err == nil {
			t.Fatalf("accepted unsafe target %d", id)
		}
	}
	if err := m.NavigateTab(context.Background(), "ai-critic", 902, "javascript:alert(1)"); err == nil {
		t.Fatal("accepted javascript URL")
	}
	if _, ok := f.last("cdp"); ok {
		t.Fatal("invalid target sent a navigation job")
	}
}

func TestNavigateTabSurfacesBrowserError(t *testing.T) {
	f := healthyFake()
	m := newTestManager(t, f)
	m.sessionID, m.contentTabID, m.url = "ai-critic", 902, "https://old.example"
	f.handle = func(call fakeCall) (json.RawMessage, bool) {
		if call.Type == "cdp" {
			return json.RawMessage(`{"result":{"errorText":"net::ERR_NAME_NOT_RESOLVED"}}`), true
		}
		return nil, false
	}
	err := m.NavigateTab(context.Background(), "ai-critic", 902, "https://invalid.example")
	if err == nil || !strings.Contains(err.Error(), "ERR_NAME_NOT_RESOLVED") {
		t.Fatalf("error=%v", err)
	}
	if m.CurrentURL() != "https://old.example" {
		t.Fatal("failed navigation replaced current URL")
	}
}

func TestTargetedNavigateAPIRequiresBothIDs(t *testing.T) {
	f := healthyFake()
	m := newTestManager(t, f)
	for _, body := range []string{`{"url":"https://example.com","session_id":"ai-critic"}`, `{"url":"https://example.com","tab_id":902}`} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/browser/navigate", strings.NewReader(body))
		m.handleNavigate(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("partial selector HTTP %d: %s", w.Code, w.Body.String())
		}
	}
}

func TestRetargetPlaceholderClearsPreviousAddress(t *testing.T) {
	f := healthyFake()
	m := newTestManager(t, f)
	m.sessionID, m.contentTabID, m.url = "ai-critic", 902, "https://before.example"
	if _, err := m.retarget(context.Background(), "ai-critic", 902); err != nil {
		t.Fatal(err)
	}
	if m.CurrentURL() != "" {
		t.Fatalf("placeholder kept old address: %q", m.CurrentURL())
	}
}

func TestPollTabsUpdatesAddressForSameTabAndPlaceholder(t *testing.T) {
	f := healthyFake()
	m := newTestManager(t, f)
	m.sessionID, m.connected, m.windowID, m.contentTabID = "ai-critic", true, 900, 902
	m.url = "https://before.example/"
	f.info = `{"window_id":900,"tabs":[{"id":902,"url":"https://after.example/path?q=1#part","role":"user","active":true}]}`
	m.pollTabs(context.Background())
	if m.CurrentURL() != "https://after.example/path?q=1#part" {
		t.Fatalf("URL after navigation: %q", m.CurrentURL())
	}
	m.mu.Lock()
	m.tabsAt = time.Time{}
	m.mu.Unlock()
	f.info = `{"window_id":900,"tabs":[{"id":902,"url":"data:text/html,placeholder","role":"user","active":true}]}`
	m.pollTabs(context.Background())
	if m.CurrentURL() != "" {
		t.Fatalf("placeholder exposed stale URL: %q", m.CurrentURL())
	}
}
