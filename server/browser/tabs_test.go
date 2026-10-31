package browser

import (
	"context"
	"encoding/json"
	"testing"
)

// tabFake models a session window with a session page plus user tabs, and keeps
// that model in sync with the cdp calls it receives, so tab ops can be asserted
// the way Chrome would answer them.
type tabFake struct {
	*fakeClient
	tabs   []TabInfo
	window int64
	// gone mirrors Chrome: with no tab left the window closed and the
	// extension answers info jobs with an error.
	gone bool
}

func newTabFake(session string, tabs ...TabInfo) *tabFake {
	tf := &tabFake{fakeClient: &fakeClient{}, tabs: tabs, window: 900}
	if session == "" {
		session = SessionID
	}
	tf.infoFor(session)
	tf.handle = func(call fakeCall) (json.RawMessage, bool) {
		switch call.Type {
		case "info":
			if tf.gone {
				return nil, false
			}
			return json.RawMessage(tf.infoJSON()), true
		case "create_tab":
			id := int64(950)
			for _, t := range tf.tabs {
				if t.TabID >= id {
					id = t.TabID + 1
				}
			}
			tf.tabs = append(tf.tabs, TabInfo{TabID: id, URL: "data:text/html,new", Active: true, Role: "user"})
			return json.RawMessage(`{"type":"create_tab","tab_id":` + itoa(id) + `}`), true
		case "cdp":
			method, _ := call.Params["method"].(string)
			params, _ := call.Params["params"].(map[string]any)
			raw, _ := params["targetId"].(string)
			var id int64
			for _, c := range raw {
				if c < '0' || c > '9' {
					id = 0
					break
				}
				id = id*10 + int64(c-'0')
			}
			switch method {
			case "Target.activateTarget":
				for i := range tf.tabs {
					tf.tabs[i].Active = tf.tabs[i].TabID == id
				}
			case "Target.closeTarget":
				kept := tf.tabs[:0]
				for _, t := range tf.tabs {
					if t.TabID != id {
						kept = append(kept, t)
					}
				}
				tf.tabs = kept
				tf.gone = len(tf.tabs) == 0
				if len(tf.tabs) > 0 {
					tf.tabs[0].Active = true
				}
			}
			return json.RawMessage(`{}`), true
		}
		return nil, false
	}
	return tf
}

// infoFor makes the info job answer with the current model for one session id.
func (tf *tabFake) infoFor(session string) {
	tf.fakeClient.handle = nil
	tf.info = ""
	tf.fakeClient.rows = []sessionRow{{
		SessionID:        session,
		Extension:        extensionInfo{Connected: true},
		SessionPageCount: 1,
	}}
}

func (tf *tabFake) infoJSON() string {
	type wireTab struct {
		ID     int64  `json:"id"`
		URL    string `json:"url"`
		Active bool   `json:"active"`
		Role   string `json:"role"`
	}
	out := struct {
		WindowID int64     `json:"window_id"`
		Tabs     []wireTab `json:"tabs"`
	}{WindowID: tf.window}
	for _, t := range tf.tabs {
		out.Tabs = append(out.Tabs, wireTab{ID: t.TabID, URL: t.URL, Active: t.Active, Role: t.Role})
	}
	data, _ := json.Marshal(out)
	return string(data)
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func sessionPageTab(id int64) TabInfo {
	return TabInfo{TabID: id, URL: "http://127.0.0.1:43799/go?session=ai-critic", Role: "session_page", Active: true}
}

func userTab(id int64, url string, active bool) TabInfo {
	return TabInfo{TabID: id, URL: url, Active: active, Role: "user"}
}

func TestJobForSecondSession(t *testing.T) {
	f := newTabFake("ai-critic")
	m := newTestManager(t, f.fakeClient)
	m.sessionID = "ai-critic"
	m.connected = true

	if _, err := m.jobIn(context.Background(), "ai-critic-2", 0, "info", nil, 0); err != nil {
		t.Fatalf("jobIn: %v", err)
	}
	call, ok := f.last("info")
	if !ok {
		t.Fatal("no info job recorded")
	}
	if call.Session != "ai-critic-2" {
		t.Fatalf("job session = %q, want ai-critic-2", call.Session)
	}

	// The default path still uses the current session, not the constant.
	m.sessionID = "research"
	if _, err := m.jobIn(context.Background(), "", 0, "info", nil, 0); err != nil {
		t.Fatalf("jobIn: %v", err)
	}
	if call, _ := f.last("info"); call.Session != "research" {
		t.Fatalf("job session = %q, want research", call.Session)
	}
}

func TestTabOpMapping(t *testing.T) {
	f := newTabFake("ai-critic", sessionPageTab(901), userTab(902, "https://example.com/a", true))
	m := newTestManager(t, f.fakeClient)
	m.sessionID = "ai-critic"
	m.connected = true

	// activate
	if _, err := m.TabOp(context.Background(), "ai-critic", TabActivate, 902, ""); err != nil {
		t.Fatalf("activate: %v", err)
	}
	call, ok := f.last("cdp")
	if !ok {
		t.Fatal("no cdp job for activate")
	}
	if call.Params["method"] != "Target.activateTarget" {
		t.Fatalf("method = %v", call.Params["method"])
	}
	if params, _ := call.Params["params"].(map[string]any); params["targetId"] != "902" {
		t.Fatalf("targetId = %v, want 902", params["targetId"])
	}

	// create
	if _, err := m.TabOp(context.Background(), "ai-critic", TabCreate, 0, "example.com/x"); err != nil {
		t.Fatalf("create: %v", err)
	}
	call, ok = f.last("create_tab")
	if !ok {
		t.Fatal("no create_tab job")
	}
	if call.Params["url"] != "https://example.com/x" {
		t.Fatalf("create url = %v, want https://example.com/x", call.Params["url"])
	}
	if call.Params["window_id"] != int64(900) {
		t.Fatalf("create window = %v, want 900", call.Params["window_id"])
	}

	// close
	if _, err := m.TabOp(context.Background(), "ai-critic", TabClose, 902, ""); err != nil {
		t.Fatalf("close: %v", err)
	}
	call, _ = f.last("cdp")
	if call.Params["method"] != "Target.closeTarget" {
		t.Fatalf("method = %v, want Target.closeTarget", call.Params["method"])
	}
}

// Window-id rail: every tab closes like any tab. The /go anchor goes over a
// CDP close (the daemon's session-aware close job would refuse it), and
// closing the last content tab re-aims the mirror at the anchor instead of
// refusing.
func TestCloseAnchorAndLastTabAllowed(t *testing.T) {
	f := newTabFake("ai-critic", sessionPageTab(901), userTab(902, "https://example.com", true))
	m := newTestManager(t, f.fakeClient)
	m.connected = true
	m.contentTabID = 902

	// Closing the /go anchor goes over CDP, not the daemon close job.
	if _, err := m.TabOp(context.Background(), "ai-critic", TabClose, 901, ""); err != nil {
		t.Fatalf("closing the anchor: %v", err)
	}
	call, _ := f.last("cdp")
	if call.Params["method"] != "Target.closeTarget" {
		t.Fatalf("method = %v, want Target.closeTarget", call.Params["method"])
	}
	if len(f.all("close_tab")) != 0 {
		t.Fatal("anchor close must not use the daemon close job")
	}
	// The anchor was active; the close re-aims at the remaining content tab.
	if got := m.contentTab(); got != 902 {
		t.Fatalf("mirror tab = %d, want the remaining content tab 902", got)
	}

	// Closing the last content tab closes the whole window in Chrome (the fake
	// mirrors that): the close must still succeed, with the reconnect owning
	// the recovery (failover to a free window) instead of an error.
	if _, err := m.TabOp(context.Background(), "ai-critic", TabClose, 902, ""); err != nil {
		t.Fatalf("closing the last tab: %v", err)
	}
	if m.contentTab() != 0 || m.windowIDValue() != 0 {
		t.Fatalf("mirror state = tab %d window %d, want cleared for the gone window", m.contentTab(), m.windowIDValue())
	}
}

func TestRetargetAfterActivate(t *testing.T) {
	f := newTabFake("ai-critic", sessionPageTab(901), userTab(902, "https://example.com/a", true), userTab(903, "https://example.com/b", false))
	m := newTestManager(t, f.fakeClient)
	m.connected = true
	m.contentTabID = 902
	m.windowID = 900
	f.vpW, f.vpH = 1200, 901

	if _, err := m.TabOp(context.Background(), "ai-critic", TabActivate, 903, ""); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if got := m.contentTab(); got != 903 {
		t.Fatalf("content tab = %d, want 903", got)
	}
	m.mu.Lock()
	url := m.url
	m.mu.Unlock()
	if url != "https://example.com/b" {
		t.Fatalf("url = %q, want https://example.com/b", url)
	}
	if _, ok := f.last("eval"); !ok {
		t.Fatal("no viewport refresh after retarget")
	}
	tabs, active := m.Tabs()
	if active != 903 {
		t.Fatalf("active tab = %d, want 903", active)
	}
	if len(tabs) != 3 {
		t.Fatalf("tabs = %d, want 3", len(tabs))
	}
}

func TestStatusCarriesTabs(t *testing.T) {
	f := newTabFake("ai-critic", sessionPageTab(901), userTab(902, "https://example.com", true))
	m := newTestManager(t, f.fakeClient)
	m.connected = true
	if _, err := m.TabOp(context.Background(), "ai-critic", TabActivate, 902, ""); err != nil {
		t.Fatalf("activate: %v", err)
	}
	st := m.Snapshot()
	if st.Session.ID != SessionID {
		t.Fatalf("session id = %q, want %q", st.Session.ID, SessionID)
	}
	if st.ActiveTabID != 902 {
		t.Fatalf("active_tab_id = %d, want 902", st.ActiveTabID)
	}
	if len(st.Tabs) != 2 || st.Tabs[1].Role != "user" {
		t.Fatalf("tabs = %+v", st.Tabs)
	}
}

func TestSessionsAggregatesTabs(t *testing.T) {
	f := newTabFake("ai-critic", sessionPageTab(901), userTab(902, "https://example.com", true))
	f.rows = []sessionRow{
		{SessionID: "ai-critic", Extension: extensionInfo{Connected: true}, SessionPageCount: 1},
		{SessionID: "ai-critic-2", Extension: extensionInfo{Connected: true}, SessionPageCount: 1},
		{SessionID: "ai-critic-1", Extension: extensionInfo{Connected: false}, SessionPageCount: 0},
	}
	m := newTestManager(t, f.fakeClient)
	m.sessionID = "ai-critic-2"
	m.connected = true
	m.windowID = 900
	m.contentTabID = 902
	m.tabs = []TabInfo{sessionPageTab(901), userTab(902, "https://example.com", true)}

	resp := m.Sessions(context.Background())
	if !resp.OK {
		t.Fatalf("sessions: %v", resp.Error)
	}
	if len(resp.Sessions) != 2 {
		t.Fatalf("sessions = %d, want 2 (dead registry rows hidden): %+v", len(resp.Sessions), resp.Sessions)
	}
	// The current row is second, not first: selection must not reorder the rail.
	if resp.Sessions[0].ID != "ai-critic" || resp.Sessions[0].Current {
		t.Fatalf("first row = %+v, want non-current ai-critic", resp.Sessions[0])
	}
	if len(resp.Sessions[0].Tabs) != 2 {
		t.Fatalf("first row tabs = %+v", resp.Sessions[0].Tabs)
	}
	if !resp.Sessions[1].Current || !resp.Sessions[1].Attachable || resp.Sessions[1].ID != "ai-critic-2" {
		t.Fatalf("second row = %+v, want current ai-critic-2", resp.Sessions[1])
	}
	if len(resp.Sessions[1].Tabs) != 2 {
		t.Fatalf("current tabs = %+v", resp.Sessions[1].Tabs)
	}

	// Switching turns the flags over but leaves the order exactly as it was.
	m.sessionID = "ai-critic"
	resp = m.Sessions(context.Background())
	if len(resp.Sessions) != 2 || resp.Sessions[0].ID != "ai-critic" || resp.Sessions[1].ID != "ai-critic-2" {
		t.Fatalf("order changed after switch: %+v", resp.Sessions)
	}
	if !resp.Sessions[0].Current || resp.Sessions[1].Current {
		t.Fatalf("current flags = %+v, want only ai-critic current", resp.Sessions)
	}
}

func TestVersionLess(t *testing.T) {
	cases := []struct {
		have, want string
		out        bool
	}{
		{"1.0.19", "1.0.20", true},
		{"1.0.20", "1.0.20", false},
		{"1.0.21", "1.0.20", false},
		{"v1.0.20", "1.0.20", false},
		{"", "1.0.20", true},
		{"dirty-build", "1.0.20", true},
	}
	for _, c := range cases {
		if got := versionLess(c.have, c.want); got != c.out {
			t.Errorf("versionLess(%q, %q) = %v, want %v", c.have, c.want, got, c.out)
		}
	}
}

func TestReadOnlyRefusesTabOp(t *testing.T) {
	f := newTabFake("ai-critic", sessionPageTab(901), userTab(902, "https://example.com", true))
	m := newTestManager(t, f.fakeClient)
	m.connected = true
	m.readOnly = true
	if _, err := m.TabOp(context.Background(), "ai-critic", TabActivate, 902, ""); err == nil {
		t.Fatal("want a read-only error")
	}
	if calls := f.all("info"); len(calls) != 0 {
		t.Fatalf("info jobs = %d, want 0: read-only must not touch the daemon", len(calls))
	}
}
