package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func confirmationFake(status string) *fakeClient {
	f := &fakeClient{}
	f.handle = func(call fakeCall) (json.RawMessage, bool) {
		switch call.Type {
		case "info":
			return json.RawMessage(`{"window_id":900,"features":["tab-close-confirm-v1"],"tabs":[{"id":1,"url":"http://127.0.0.1:43761/go?session=background","role":"session_page"},{"id":2,"url":"https://example.com/a","role":"user"},{"id":3,"url":"https://example.com/b","role":"user"}]}`), true
		case "cdp":
			if call.Params["method"] == "BrowserAgent.closeTab" {
				return json.RawMessage(`{"result":{"status":"confirmation_required","operation_id":"op-1","tab_id":2}}`), true
			}
			if call.Params["method"] == "BrowserAgent.resolveCloseTab" {
				return json.RawMessage(fmt.Sprintf(`{"result":{"status":%q,"tab_id":2}}`, status)), true
			}
		}
		return nil, false
	}
	return f
}

func TestRemoteTabCloseRequiresExplicitDecision(t *testing.T) {
	f := confirmationFake("kept")
	m := newTestManager(t, f)
	m.sessionID = "current"
	m.connected = true
	info, err := m.TabOp(context.Background(), "background", TabClose, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if info.CloseConfirmation == nil || info.CloseConfirmation.OperationID != "op-1" || info.CloseConfirmation.TabID != 2 {
		t.Fatalf("confirmation = %#v", info.CloseConfirmation)
	}
	call, _ := f.last("cdp")
	if call.Session != "background" || call.Params["method"] != "BrowserAgent.closeTab" {
		t.Fatalf("start call = %#v", call)
	}
	out, err := m.ResolveTabClose(context.Background(), "background", "op-1", false)
	if err != nil {
		t.Fatal(err)
	}
	if out.CloseOutcome != "kept" {
		t.Fatalf("outcome = %q", out.CloseOutcome)
	}
	call, _ = f.last("cdp")
	params := call.Params["params"].(map[string]any)
	if call.Session != "background" || call.Params["method"] != "BrowserAgent.resolveCloseTab" || params["accept"] != false || params["operation_id"] != "op-1" {
		t.Fatalf("decision call = %#v", call)
	}
}

func TestOldExtensionTimeoutReportsUnknownWithoutRetry(t *testing.T) {
	f := newTabFake("background", sessionPageTab(1), userTab(2, "https://example.com/a", true), userTab(3, "https://example.com/b", false))
	f.fail = func(call fakeCall) error {
		if call.Type == "cdp" && call.Params["method"] == "Target.closeTarget" {
			return fmt.Errorf("timeout waiting for job result")
		}
		return nil
	}
	m := newTestManager(t, f.fakeClient)
	m.sessionID = "current"
	m.connected = true
	_, err := m.TabOp(context.Background(), "background", TabClose, 2, "")
	if err == nil || !strings.Contains(err.Error(), "outcome is unknown") {
		t.Fatalf("error = %v", err)
	}
	count := 0
	for _, call := range f.calls {
		if call.Type == "cdp" && call.Params["method"] == "Target.closeTarget" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("close calls = %d, want exactly one", count)
	}
}

func TestRemoteTabCloseUnknownIsNotSuccess(t *testing.T) {
	f := confirmationFake("unknown")
	m := newTestManager(t, f)
	m.sessionID = "current"
	m.connected = true
	_, err := m.ResolveTabClose(context.Background(), "background", "op-1", true)
	if err == nil || !strings.Contains(err.Error(), "outcome is unknown") {
		t.Fatalf("error = %v", err)
	}
}

func TestConfirmCloseRequiresExplicitBoolean(t *testing.T) {
	f := confirmationFake("kept")
	m := newTestManager(t, f)
	m.sessionID = "current"
	m.connected = true
	for _, body := range []string{`{"op":"confirm_close","session_id":"background","operation_id":"op-1"}`, `{"op":"confirm_close","session_id":"background","accept":false}`} {
		req := httptest.NewRequest(http.MethodPost, "/api/browser/tab", strings.NewReader(body))
		w := httptest.NewRecorder()
		m.handleTab(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("body %s: status %d", body, w.Code)
		}
	}
}

func TestConfirmCloseFalseIsNotTreatedAsMissing(t *testing.T) {
	f := confirmationFake("kept")
	m := newTestManager(t, f)
	m.sessionID = "current"
	m.connected = true
	req := httptest.NewRequest(http.MethodPost, "/api/browser/tab", strings.NewReader(`{"op":"confirm_close","session_id":"background","operation_id":"op-1","accept":false}`))
	w := httptest.NewRecorder()
	m.handleTab(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"close_outcome":"kept"`) {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
}
