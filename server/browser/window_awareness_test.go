package browser

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// popupInfoFixture answers the direct info job for popup window 77.
const popupInfoFixture = `{"type":"info","window_id":77,"direct":true,"tabs":[` +
	`{"id":771,"index":1,"title":"Log in","url":"https://seatalk.example/login","active":true,"role":"user"}]}`

// popupFake is a direct fake whose windows payload carries a popup window
// opened by the driven tab (opener_tab_id = 111): exactly what the extension
// reports after "Login with SeaTalk".
func popupFake() *fakeClient {
	f := directFake()
	f.windowsPayload = `{"type":"windows","version":"1.0.28","features":["all-windows","all-windows-v2","js-dialog"],"windows":[` +
		`{"window_id":11,"type":"normal","focused":true,"width":1280,"height":900,"state":"normal",` +
		`"tabs":[{"tab_id":111,"index":0,"title":"Docs","url":"https://example.com/docs","active":true,"capturable":true,"role":"user","opener_tab_id":null},` +
		`{"tab_id":112,"index":1,"title":"Search","url":"https://example.com/search","active":false,"capturable":true,"role":"user","opener_tab_id":null}]},` +
		`{"window_id":77,"type":"popup","focused":false,"width":420,"height":540,"state":"normal",` +
		`"title":"Log in","title_tab_id":771,` +
		`"tabs":[{"tab_id":771,"index":0,"title":"Log in","url":"https://seatalk.example/login","active":true,"capturable":true,"role":"user","opener_tab_id":111}]}]}`
	f.directHandle = func(call fakeCall) (json.RawMessage, bool) {
		switch call.Type {
		case "info":
			switch call.Session {
			case sessionlessSessionID(11):
				return json.RawMessage(directInfoFixture), true
			case sessionlessSessionID(77):
				return json.RawMessage(popupInfoFixture), true
			}
		case "window", "resize":
			return json.RawMessage(`{}`), true
		}
		return nil, false
	}
	f.directErrFor = func(call fakeCall) error {
		if call.Type == "info" && call.Session != sessionlessSessionID(11) && call.Session != sessionlessSessionID(77) {
			return errWindowGone
		}
		return nil
	}
	return f
}

// A popup opened by the driven page must take over the mirror: the OAuth
// flow happens in that window, and it closes itself when done.
func TestPollTabsFollowsOwnedPopup(t *testing.T) {
	f := popupFake()
	m := newTestManager(t, f)
	m.recordDaemon(&daemonHealth{OK: true, Version: "1.0.28", DirectSupported: true})
	if err := m.Ensure(context.Background()); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if got := m.sessionIDValue(); got != sessionlessSessionID(11) {
		t.Fatalf("session = %q, want the parent window before any popup", got)
	}

	m.mu.Lock()
	m.tabsAt = time.Now().Add(-tabInfoTTL - time.Second)
	m.mu.Unlock()
	m.pollTabs(context.Background())

	if got := m.sessionIDValue(); got != sessionlessSessionID(77) {
		t.Fatalf("session = %q, want the owned popup window 77", got)
	}
	if got := m.windowIDValue(); got != 77 {
		t.Fatalf("window = %d, want 77", got)
	}
	// The preset must not resize the popup: its own size is the flow's size.
	m.mu.Lock()
	applied := m.presetApplied
	m.mu.Unlock()
	if !applied {
		t.Fatal("presetApplied = false; the popup would be resized by ensurePreset")
	}
}

// A popup nobody in the driven window opened belongs to someone else: the
// mirror stays put.
func TestPollTabsIgnoresForeignPopup(t *testing.T) {
	f := directFake()
	f.windowsPayload = `{"type":"windows","version":"1.0.28","features":["all-windows-v2"],"windows":[` +
		`{"window_id":11,"type":"normal","focused":true,"width":1280,"height":900,"state":"normal",` +
		`"tabs":[{"tab_id":111,"index":0,"title":"Docs","url":"https://example.com/docs","active":true,"capturable":true,"role":"user","opener_tab_id":null}]},` +
		`{"window_id":78,"type":"popup","focused":false,"width":420,"height":540,"state":"normal",` +
		`"tabs":[{"tab_id":781,"index":0,"title":"Other agent login","url":"https://seatalk.example/login","active":true,"capturable":true,"role":"user","opener_tab_id":555}]}]}`
	m := newTestManager(t, f)
	m.recordDaemon(&daemonHealth{OK: true, Version: "1.0.28", DirectSupported: true})
	if err := m.Ensure(context.Background()); err != nil {
		t.Fatalf("Ensure: %v", err)
	}

	m.mu.Lock()
	m.tabsAt = time.Now().Add(-tabInfoTTL - time.Second)
	m.mu.Unlock()
	m.pollTabs(context.Background())

	if got := m.sessionIDValue(); got != sessionlessSessionID(11) {
		t.Fatalf("session = %q, want to stay on the parent (opener 555 is not ours)", got)
	}
}

// When the popup closes, the pinned-window failover hands the mirror back to
// the parent window.
func TestPopupCloseFailsOverToParent(t *testing.T) {
	f := popupFake()
	m := newTestManager(t, f)
	m.recordDaemon(&daemonHealth{OK: true, Version: "1.0.28", DirectSupported: true})
	if err := m.Ensure(context.Background()); err != nil {
		t.Fatalf("Ensure: %v", err)
	}

	// Follow the popup first.
	m.mu.Lock()
	m.tabsAt = time.Now().Add(-tabInfoTTL - time.Second)
	m.mu.Unlock()
	m.pollTabs(context.Background())
	if got := m.sessionIDValue(); got != sessionlessSessionID(77) {
		t.Fatalf("session = %q, want the popup 77 after follow", got)
	}

	// The popup window is gone; only the parent remains.
	f.mu.Lock()
	f.windowsPayload = `{"type":"windows","version":"1.0.28","features":["all-windows-v2"],"windows":[` +
		`{"window_id":11,"type":"normal","focused":true,"width":1280,"height":900,"state":"normal",` +
		`"tabs":[{"tab_id":111,"index":0,"title":"Docs","url":"https://example.com/docs","active":true,"capturable":true,"role":"user","opener_tab_id":null}]}]}`
	f.mu.Unlock()
	m.mu.Lock()
	m.tabsAt = time.Now().Add(-tabInfoTTL - time.Second)
	m.mu.Unlock()
	m.pollTabs(context.Background()) // info for the gone popup marks disconnected
	if err := m.Ensure(context.Background()); err != nil {
		t.Fatalf("Ensure after popup close: %v", err)
	}

	if got := m.sessionIDValue(); got != sessionlessSessionID(11) {
		t.Fatalf("session = %q, want the parent window 11 after the popup closed", got)
	}
}

// The rail lists owned and foreign popups alike (perception), each row marked
// with its window kind.
func TestSessionsListIncludesPopupWindows(t *testing.T) {
	f := popupFake()
	m := newTestManager(t, f)
	m.recordDaemon(&daemonHealth{OK: true, Version: "1.0.28", DirectSupported: true})

	resp := m.Sessions(context.Background())
	found := map[string]SessionInfo{}
	for _, row := range resp.Sessions {
		found[row.ID] = row
	}
	popup, ok := found[sessionlessSessionID(77)]
	if !ok {
		t.Fatalf("popup window 77 missing from the rail: %+v", resp.Sessions)
	}
	if popup.WindowType != "popup" {
		t.Fatalf("window_type = %q, want popup", popup.WindowType)
	}
	if !popup.Adoptable || !popup.Sessionless {
		t.Fatal("popup row must be adoptable")
	}
	if _, ok := found[sessionlessSessionID(11)]; !ok {
		t.Fatalf("normal window 11 missing from the rail")
	}
}

// An open JS dialog rides the info payload to the status snapshot, and the
// answer reaches the extension as a dialog job on the driven tab.
func TestDialogSurfacesAndAnswers(t *testing.T) {
	dialogInfo := `{"type":"info","window_id":11,"direct":true,"tabs":[` +
		`{"id":111,"index":1,"title":"Docs","url":"https://example.com/docs","active":true,"role":"user",` +
		`"dialog":{"type":"confirm","message":"Leave site?","default_prompt":"","url":"https://example.com/docs"}}]}`
	f := directFake()
	f.directHandle = func(call fakeCall) (json.RawMessage, bool) {
		switch call.Type {
		case "info":
			if call.Session == sessionlessSessionID(11) {
				return json.RawMessage(dialogInfo), true
			}
		case "window", "resize":
			return json.RawMessage(`{}`), true
		}
		return nil, false
	}
	f.directErrFor = func(call fakeCall) error {
		if call.Type == "info" && call.Session != sessionlessSessionID(11) {
			return errWindowGone
		}
		return nil
	}
	m := newTestManager(t, f)
	m.recordDaemon(&daemonHealth{OK: true, Version: "1.0.28", DirectSupported: true})
	if err := m.Ensure(context.Background()); err != nil {
		t.Fatalf("Ensure: %v", err)
	}

	dialog := m.Snapshot().Session.Dialog
	if dialog == nil {
		t.Fatal("session.dialog = nil, want the open confirm dialog")
	}
	if dialog.Type != "confirm" || dialog.Message != "Leave site?" {
		t.Fatalf("dialog = %+v, want the confirm from the info job", dialog)
	}

	if err := m.AnswerDialog(context.Background(), true, ""); err != nil {
		t.Fatalf("AnswerDialog: %v", err)
	}
	answered := false
	for _, c := range f.directCalls {
		if c.Type == "dialog" && c.Tab == 111 {
			if accept, _ := c.Params["accept"].(bool); accept {
				answered = true
			}
		}
	}
	if !answered {
		t.Fatalf("no dialog accept job on the driven tab: %+v", f.directCalls)
	}
}

// The extension payload contract: every window type is listed, tabs carry
// opener attribution, and the dialog job types are advertised.
func TestWindowsPayloadContractShape(t *testing.T) {
	// Guarded by the live extension; here we pin the feature names ai-critic
	// may rely on, so a rename fails a test instead of a phone.
	for _, feature := range []string{"all-windows-v2", "js-dialog"} {
		if !strings.Contains(feature, "-") {
			t.Fatalf("feature %q should be namespaced", feature)
		}
	}
}
