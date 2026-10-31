package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestWindowTitleFromTabs(t *testing.T) {
	tests := []struct {
		name      string
		tabs      []TabInfo
		wantTitle string
		wantTab   int64
	}{
		{
			name: "lowest real page wins over the active tab",
			tabs: []TabInfo{
				{TabID: 4, Index: 3, Title: "Active tab", URL: "https://active.example", Active: true, Role: "user"},
				{TabID: 2, Index: 1, Title: "  Confluence\n — AI   Critic ", URL: "https://confluence.example", Role: "user"},
				{TabID: 1, Index: 0, URL: "http://127.0.0.1:23712/go?session=s1", Role: "session_page"},
			},
			wantTitle: "Confluence — AI Critic",
			wantTab:   2,
		},
		{
			name: "first page skips chrome tabs",
			tabs: []TabInfo{
				{TabID: 1, Index: 0, Title: "New Tab", URL: "chrome://newtab", Role: "user"},
				{TabID: 2, Index: 1, Title: "GitHub", URL: "https://github.com", Role: "user"},
			},
			wantTitle: "GitHub",
			wantTab:   2,
		},
		{
			name: "chrome pages fall back to their titles",
			tabs: []TabInfo{
				{TabID: 1, Index: 0, Title: "New Tab", URL: "chrome://newtab", Role: "user"},
				{TabID: 2, Index: 1, Title: "Settings", URL: "chrome://settings", Role: "user"},
			},
			wantTitle: "New Tab",
			wantTab:   1,
		},
		{
			name: "settings only",
			tabs: []TabInfo{
				{TabID: 2, Index: 0, Title: "Settings", URL: "chrome://settings", Role: "user"},
			},
			wantTitle: "Settings",
			wantTab:   2,
		},
		{
			name: "only a session page uses its id",
			tabs: []TabInfo{
				{TabID: 1, Index: 0, Title: "browser-agent", URL: "http://127.0.0.1:23712/go?session=sess-7lm3dk", Role: "session_page"},
			},
			wantTitle: "sess-7lm3dk",
			wantTab:   0,
		},
		{
			name:      "no tabs",
			wantTitle: "Untitled window",
		},
		{
			name: "empty page title falls back to the host",
			tabs: []TabInfo{
				{TabID: 1, Index: 0, URL: "https://fallback.example/path", Role: "user"},
			},
			wantTitle: "fallback.example",
			wantTab:   1,
		},
		{
			name: "file title falls back to the basename",
			tabs: []TabInfo{
				{TabID: 1, Index: 0, URL: "file:///tmp/report.pdf", Role: "user"},
			},
			wantTitle: "report.pdf",
			wantTab:   1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, tabID := windowTitleFromTabs(tt.tabs)
			if title != tt.wantTitle || tabID != tt.wantTab {
				t.Fatalf("title = %q, tab = %d; want %q, %d", title, tabID, tt.wantTitle, tt.wantTab)
			}
		})
	}
}

func TestWindowLabelPrefersExtensionTitle(t *testing.T) {
	win := &windowSnapshot{
		Title:      "  Design   docs ",
		TitleTabID: 12,
		Tabs: []windowTab{
			{TabID: 11, Index: 0, Title: "Other", URL: "https://other.example"},
			{TabID: 12, Index: 1, Title: "Design docs", URL: "https://design.example"},
		},
	}
	if title, tabID := windowLabel(win); title != "Design docs" || tabID != 12 {
		t.Fatalf("windowLabel = %q, %d; want Design docs, 12", title, tabID)
	}
}

func TestMergeWindowsKeepsSessionsAndAddsOnlyUnknownWindows(t *testing.T) {
	payload := &windowsPayload{
		Features: []string{"all-windows"},
		Windows: []windowSnapshot{
			{
				WindowID:   101,
				Title:      "Confluence — AI Critic",
				SessionIDs: []string{"s1", "s2"},
				Tabs: []windowTab{
					{TabID: 1, Index: 0, Title: "Confluence — AI Critic", URL: "https://confluence.example", Active: true, Role: "user"},
					{TabID: 2, Index: 1, Title: "browser-agent", URL: "http://127.0.0.1:23712/go?session=s1", Role: "session"},
				},
			},
			{
				WindowID: 102,
				Title:    "Design docs — Notion",
				Tabs: []windowTab{
					{TabID: 21, Index: 0, Title: "Design docs — Notion", URL: "https://notion.example", Active: true, Role: "user"},
					{TabID: 22, Index: 1, Title: "Google", URL: "https://google.example", Role: "user"},
				},
			},
			{
				WindowID:   103,
				SessionIDs: []string{"stale"},
				Tabs: []windowTab{
					{TabID: 31, Index: 0, Title: "browser-agent", URL: "http://127.0.0.1:23712/go?session=stale", Role: "session"},
				},
			},
		},
	}
	resp := SessionsResponse{Sessions: []SessionInfo{
		{ID: "s1", WindowID: 101, Title: "old title", Tabs: []TabInfo{{TabID: 1, Index: 0, Title: "cached", URL: "https://cached.example"}}},
		{ID: "s2", WindowID: 101},
		{ID: "s3", WindowID: 999, Title: "Gone window"},
		{ID: "s4", Title: "Info title"},
	}}
	m := &Manager{}
	m.mergeWindows(&resp, payload)

	if len(resp.Sessions) != 6 {
		t.Fatalf("sessions = %d, want 6: %+v", len(resp.Sessions), resp.Sessions)
	}
	byID := map[string]SessionInfo{}
	for _, row := range resp.Sessions {
		byID[row.ID] = row
	}
	if got := byID["s1"].Title; got != "Confluence — AI Critic" {
		t.Fatalf("s1 title = %q", got)
	}
	if got := byID["s1"].Tabs; len(got) != 1 || got[0].Title != "cached" {
		t.Fatalf("s1 tabs were replaced: %+v", got)
	}
	if got := byID["s2"].Title; got != "Confluence — AI Critic" {
		t.Fatalf("s2 title = %q", got)
	}
	if got := byID["s2"].Tabs; len(got) != 2 {
		t.Fatalf("s2 tabs = %+v, want the window tabs", got)
	}
	if got := byID["s3"].Title; got != "Gone window" {
		t.Fatalf("unmatched session title changed to %q", got)
	}
	if got := byID["s4"].Title; got != "Info title" {
		t.Fatalf("unmatched info title changed to %q", got)
	}
	discovered := byID["sess-window-102"]
	if !discovered.Sessionless || !discovered.Adoptable || discovered.Attached || discovered.Attachable {
		t.Fatalf("discovered row flags = %+v", discovered)
	}
	if discovered.Title != "Design docs — Notion" || discovered.ActiveTabID != 21 || discovered.WindowID != 102 {
		t.Fatalf("discovered row = %+v", discovered)
	}
	if !strings.Contains(discovered.URL, "notion.example") {
		t.Fatalf("discovered URL = %q, want the active content tab", discovered.URL)
	}
	if got := byID["sess-window-103"].Title; got != "stale" {
		t.Fatalf("stale /go window title = %q, want its session id", got)
	}
	if _, ok := byID["sess-window-101"]; ok {
		t.Fatal("a live session window was duplicated as sessionless")
	}
}

func attachFake(windowsJSON string) *fakeClient {
	f := &fakeClient{rows: []sessionRow{
		{SessionID: "s1", Extension: extensionInfo{Connected: true}, SessionPageCount: 1},
	}}
	f.handle = func(call fakeCall) (json.RawMessage, bool) {
		switch call.Type {
		case "info":
			switch call.Session {
			case "s1":
				return json.RawMessage(`{"window_id":900,"tabs":[` +
					`{"id":901,"url":"http://127.0.0.1:23712/go?session=s1","active":false,"role":"session_page"},` +
					`{"id":902,"url":"https://anchor.example","title":"Anchor","active":true,"role":"user"}]}`), true
			case "sess-window-950":
				return json.RawMessage(`{"window_id":950,"tabs":[` +
					`{"id":951,"url":"http://127.0.0.1:23712/go?session=sess-window-950","active":false,"role":"session_page"},` +
					`{"id":952,"url":"https://design.example","title":"Design docs","active":true,"role":"user"}]}`), true
			}
		case "windows":
			return json.RawMessage(windowsJSON), true
		case "create_tab":
			return json.RawMessage(`{"tab_id":951,"window_id":950}`), true
		}
		return nil, false
	}
	return f
}

func TestAttachWindowAdoptsExistingWindow(t *testing.T) {
	f := attachFake(`{"type":"windows","version":"1.0.21","features":["all-windows"],"windows":[` +
		`{"window_id":950,"title":"Design docs","tabs":[` +
		`{"tab_id":952,"index":0,"title":"Design docs","url":"https://design.example","active":true,"role":"user"}]}]}`)
	m := newTestManager(t, f)
	m.sessionID = "s1"
	m.connected = true
	m.windowID = 900
	m.contentTabID = 902
	m.tabs = []TabInfo{sessionPageTab(901), userTab(902, "https://anchor.example", true)}
	m.url = "https://anchor.example"
	if err := saveConfig(Config{Mode: ModeCast, Preset: "mac", SessionID: "s1"}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	id, err := m.AttachWindow(ctx, 950, "sess-window-950")
	if err != nil {
		t.Fatalf("AttachWindow: %v", err)
	}
	if id != "sess-window-950" {
		t.Fatalf("session id = %q", id)
	}
	creates := f.all("create_tab")
	if len(creates) != 1 {
		t.Fatalf("create_tab calls = %d, want one", len(creates))
	}
	f.mu.Lock()
	registered := append([]string(nil), f.created...)
	f.mu.Unlock()
	if len(registered) == 0 {
		t.Fatalf("adopted session was never registered")
	}
	for _, got := range registered {
		if got != id {
			t.Fatalf("registered sessions = %v, want %s", registered, id)
		}
	}
	call := creates[0]
	if got := fmt.Sprint(call.Params["window_id"]); got != "950" {
		t.Fatalf("create_tab window = %s, want 950", got)
	}
	if got := fmt.Sprint(call.Params["url"]); got != m.sessionPageURL(id) {
		t.Fatalf("create_tab url = %s", got)
	}
	if m.sessionIDValue() != id || m.windowIDValue() != 950 || m.contentTab() != 952 {
		t.Fatalf("manager did not adopt the window: session=%q window=%d content=%d", m.sessionIDValue(), m.windowIDValue(), m.contentTab())
	}
	if got := loadConfig().SessionID; got != "s1" {
		t.Fatalf("saved session = %q, want s1 (synthetic ids are ephemeral)", got)
	}
	if got := m.newWindowFor; got != "" {
		t.Fatalf("newWindowFor = %q, want no new Chrome window", got)
	}

	// Adoption takes ensureMu to switch sessions. It must release it, or every
	// later Ensure (status refresh, preset change, viewport measure) hangs.
	done := make(chan error, 1)
	go func() { done <- m.Ensure(ctx) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Ensure after attach: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Ensure after attach blocked; ensureMu was not released")
	}
}

func TestAttachWindowReusesExistingSessionPage(t *testing.T) {
	f := attachFake(`{"features":["all-windows"],"windows":[` +
		`{"window_id":950,"title":"Design docs","session_ids":["sess-window-950"],"tabs":[` +
		`{"tab_id":951,"index":0,"url":"http://127.0.0.1:23712/go?session=sess-window-950","role":"session"},` +
		`{"tab_id":952,"index":1,"title":"Design docs","url":"https://design.example","active":true,"role":"user"}]}]}`)
	m := newTestManager(t, f)
	m.sessionID = "s1"
	m.connected = true
	m.windowID = 900
	m.contentTabID = 902

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := m.AttachWindow(ctx, 950, ""); err != nil {
		t.Fatalf("AttachWindow: %v", err)
	}
	if creates := f.all("create_tab"); len(creates) != 0 {
		t.Fatalf("create_tab calls = %d, want zero on retry", len(creates))
	}
}

func TestAttachWindowRequiresFeature(t *testing.T) {
	f := attachFake(`{"features":["browser-agent"],"windows":[` +
		`{"window_id":950,"title":"Design docs","tabs":[]}]}`)
	m := newTestManager(t, f)
	m.sessionID = "s1"
	m.connected = true
	m.windowID = 900
	m.contentTabID = 902

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := m.AttachWindow(ctx, 950, "sess-window-950")
	if !errors.Is(err, errAllWindowsUnsupported) {
		t.Fatalf("err = %v, want unsupported", err)
	}
	if creates := f.all("create_tab"); len(creates) != 0 {
		t.Fatalf("create_tab calls = %d, want zero", len(creates))
	}
	if m.sessionIDValue() != "s1" {
		t.Fatalf("session changed to %q", m.sessionIDValue())
	}
}

func TestAttachWindowRefusesWindowWithLiveSession(t *testing.T) {
	f := attachFake(`{"features":["all-windows"],"windows":[` +
		`{"window_id":950,"session_ids":["other"],"tabs":[` +
		`{"tab_id":951,"index":0,"url":"http://127.0.0.1:23712/go?session=other","role":"session"}]}]}`)
	m := newTestManager(t, f)
	m.sessionID = "s1"
	m.connected = true
	m.windowID = 900
	m.contentTabID = 902

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := m.AttachWindow(ctx, 950, "sess-window-950")
	if !errors.Is(err, errWindowHasSession) {
		t.Fatalf("err = %v, want window-has-session", err)
	}
	if creates := f.all("create_tab"); len(creates) != 0 {
		t.Fatalf("create_tab calls = %d, want zero", len(creates))
	}
}

func TestAttachWindowRejectsRegistrationInAnotherWindow(t *testing.T) {
	f := attachFake(`{"features":["all-windows"],"windows":[` +
		`{"window_id":950,"title":"Design docs","tabs":[` +
		`{"tab_id":952,"index":0,"title":"Design docs","url":"https://design.example","active":true,"role":"user"}]}]}`)
	// The old extension ignores create_tab.window_id and opens /go in the
	// anchor window. The server must notice rather than silently driving the
	// wrong window.
	f.handle = func(call fakeCall) (json.RawMessage, bool) {
		switch call.Type {
		case "info":
			return json.RawMessage(`{"window_id":900,"tabs":[` +
				`{"id":901,"url":"http://127.0.0.1:23712/go?session=sess-window-950","active":true}]}`), true
		case "windows":
			return json.RawMessage(`{"features":["all-windows"],"windows":[` +
				`{"window_id":950,"title":"Design docs","tabs":[` +
				`{"tab_id":952,"index":0,"title":"Design docs","url":"https://design.example","active":true,"role":"user"}]}]}`), true
		case "create_tab":
			return json.RawMessage(`{"tab_id":901,"window_id":900}`), true
		}
		return nil, false
	}
	m := newTestManager(t, f)
	m.sessionID = "s1"
	m.connected = true
	m.windowID = 900
	m.contentTabID = 902
	if err := saveConfig(Config{Mode: ModeCast, Preset: "mac", SessionID: "s1"}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := m.AttachWindow(ctx, 950, "sess-window-950")
	if err == nil || !strings.Contains(err.Error(), "registered in window 900") {
		t.Fatalf("err = %v, want a wrong-window failure", err)
	}
	if m.sessionIDValue() != "s1" || m.windowIDValue() != 900 || m.contentTab() != 902 {
		t.Fatalf("manager changed on failure: session=%q window=%d content=%d", m.sessionIDValue(), m.windowIDValue(), m.contentTab())
	}
	if got := loadConfig().SessionID; got != "s1" {
		t.Fatalf("saved session = %q, want s1", got)
	}
}

func TestAttachWindowRollsBackWhenSwitchFails(t *testing.T) {
	f := attachFake(`{"features":["all-windows"],"windows":[` +
		`{"window_id":950,"title":"Design docs","tabs":[` +
		`{"tab_id":952,"index":0,"title":"Design docs","url":"https://design.example","active":true,"role":"user"}]}]}`)
	infoCalls := 0
	base := f.handle
	f.handle = func(call fakeCall) (json.RawMessage, bool) {
		if call.Type == "info" && call.Session == "sess-window-950" {
			infoCalls++
			if infoCalls > 1 {
				return json.RawMessage(`not-json`), true
			}
		}
		return base(call)
	}
	m := newTestManager(t, f)
	m.sessionID = "s1"
	m.connected = true
	m.windowID = 900
	m.contentTabID = 902
	m.tabs = []TabInfo{sessionPageTab(901), userTab(902, "https://anchor.example", true)}
	m.url = "https://anchor.example"
	if err := saveConfig(Config{Mode: ModeCast, Preset: "mac", SessionID: "s1"}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := m.AttachWindow(ctx, 950, "sess-window-950"); err == nil {
		t.Fatal("want switch failure")
	}
	if m.sessionIDValue() != "s1" || m.windowIDValue() != 900 || m.contentTab() != 902 || len(m.tabs) != 2 {
		t.Fatalf("state not rolled back: session=%q window=%d content=%d tabs=%+v", m.sessionIDValue(), m.windowIDValue(), m.contentTab(), m.tabs)
	}
	if got := loadConfig().SessionID; got != "s1" {
		t.Fatalf("saved session = %q, want s1", got)
	}
}

func TestSessionsMergesDiscoveredWindows(t *testing.T) {
	f := &fakeClient{
		rows: []sessionRow{
			{SessionID: "s1", Extension: extensionInfo{Connected: true}, SessionPageCount: 1},
			{SessionID: "s2", Extension: extensionInfo{Connected: true}, SessionPageCount: 1},
		},
	}
	f.handle = func(call fakeCall) (json.RawMessage, bool) {
		switch call.Type {
		case "info":
			switch call.Session {
			case "s1":
				return json.RawMessage(`{"window_id":900,"tabs":[` +
					`{"id":901,"index":1,"url":"http://127.0.0.1:23712/go?session=s1","role":"session_page"},` +
					`{"id":902,"index":2,"title":"Confluence — AI Critic","url":"https://confluence.example","active":true,"role":"user"}]}`), true
			case "s2":
				return json.RawMessage(`{"window_id":900,"tabs":[` +
					`{"id":901,"index":1,"url":"http://127.0.0.1:23712/go?session=s2","role":"session_page"},` +
					`{"id":903,"index":2,"title":"GitHub","url":"https://github.com","active":true,"role":"user"}]}`), true
			}
		case "windows":
			return json.RawMessage(`{"type":"windows","version":"1.0.21","features":["browser-agent","all-windows"],"windows":[` +
				`{"window_id":900,"title":"Confluence — AI Critic","session_ids":["s1"],"tabs":[]},` +
				`{"window_id":950,"title":"Design docs — Notion","tabs":[` +
				`{"tab_id":951,"index":0,"title":"Design docs — Notion","url":"https://notion.example","active":true,"role":"user"}]},` +
				`{"window_id":951,"session_ids":["stale"],"tabs":[` +
				`{"tab_id":961,"index":0,"url":"http://127.0.0.1:23712/go?session=stale","role":"session"}]}]}`), true
		}
		return nil, false
	}
	m := newTestManager(t, f)
	m.sessionID = "s1"
	m.connected = true
	m.windowID = 900
	m.contentTabID = 902
	m.tabs = []TabInfo{sessionPageTab(901), userTab(902, "https://confluence.example", true)}

	resp := m.Sessions(context.Background())
	if !resp.OK || !resp.WindowsSupported {
		t.Fatalf("response = %+v", resp)
	}
	if len(resp.Sessions) != 4 {
		t.Fatalf("sessions = %d, want 4: %+v", len(resp.Sessions), resp.Sessions)
	}
	byID := map[string]SessionInfo{}
	for _, row := range resp.Sessions {
		byID[row.ID] = row
	}
	if got := byID["s1"].Title; got != "Confluence — AI Critic" {
		t.Fatalf("s1 title = %q", got)
	}
	if got := byID["s2"].Title; got != "Confluence — AI Critic" {
		t.Fatalf("s2 title = %q", got)
	}
	discovered := byID["sess-window-950"]
	if !discovered.Sessionless || !discovered.Adoptable || discovered.Title != "Design docs — Notion" {
		t.Fatalf("discovered window = %+v", discovered)
	}
	if stale := byID["sess-window-951"]; stale.Title != "stale" || !stale.Sessionless {
		t.Fatalf("stale /go window = %+v", stale)
	}
	if got := resp.Sessions[0].ID; got != "s1" {
		t.Fatalf("first row = %q, want stable id order", got)
	}
	if calls := f.all("windows"); len(calls) != 1 || calls[0].Session != "s1" {
		t.Fatalf("windows calls = %+v", calls)
	}
}

func TestSessionsSurvivesWindowsFailure(t *testing.T) {
	f := &fakeClient{rows: []sessionRow{
		{SessionID: "s1", Extension: extensionInfo{Connected: true}, SessionPageCount: 1},
	}}
	f.fail = func(call fakeCall) error {
		if call.Type == "windows" {
			return errors.New("extension disappeared")
		}
		return nil
	}
	f.info = `{"window_id":900,"tabs":[{"id":902,"index":0,"title":"Cached page","url":"https://cached.example","active":true,"role":"user"}]}`
	m := newTestManager(t, f)
	m.sessionID = "s1"
	m.connected = true
	m.windowID = 900
	m.contentTabID = 902
	m.tabs = []TabInfo{userTab(902, "https://cached.example", true)}

	resp := m.Sessions(context.Background())
	if !resp.OK || resp.WindowsSupported {
		t.Fatalf("response = %+v", resp)
	}
	if len(resp.Sessions) != 1 || resp.Sessions[0].Title != "Cached page" {
		t.Fatalf("sessions = %+v, want the live session with its title", resp.Sessions)
	}
}

func TestSessionsSkipsDiscoveryForOldExtension(t *testing.T) {
	f := &fakeClient{rows: []sessionRow{
		{SessionID: "s1", Extension: extensionInfo{Connected: true}, SessionPageCount: 1},
	}}
	f.handle = func(call fakeCall) (json.RawMessage, bool) {
		switch call.Type {
		case "info":
			return json.RawMessage(`{"window_id":900,"tabs":[{"id":902,"index":0,"title":"Cached page","url":"https://cached.example","active":true,"role":"user"}]}`), true
		case "windows":
			return json.RawMessage(`{"features":["browser-agent"],"windows":[{"window_id":950,"title":"Design docs"}]}`), true
		}
		return nil, false
	}
	m := newTestManager(t, f)
	m.sessionID = "s1"
	m.connected = true
	m.windowID = 900
	m.contentTabID = 902
	m.tabs = []TabInfo{userTab(902, "https://cached.example", true)}

	resp := m.Sessions(context.Background())
	if resp.WindowsSupported {
		t.Fatal("old extension advertised windows support")
	}
	if len(resp.Sessions) != 1 || resp.Sessions[0].ID != "s1" {
		t.Fatalf("sessions = %+v, want no discovered rows", resp.Sessions)
	}
}

func TestSessionsReadOnlyDoesNotTouchDaemon(t *testing.T) {
	f := &fakeClient{rows: []sessionRow{
		{SessionID: "s1", Extension: extensionInfo{Connected: true}, SessionPageCount: 1},
	}}
	m := newTestManager(t, f)
	m.sessionID = "s1"
	m.connected = true
	m.readOnly = true
	m.tabs = []TabInfo{userTab(902, "https://cached.example", true)}
	m.tabs[0].Title = "Cached page"

	resp := m.Sessions(context.Background())
	if !resp.OK || resp.WindowsSupported {
		t.Fatalf("response = %+v", resp)
	}
	if len(resp.Sessions) != 1 || resp.Sessions[0].Title != "Cached page" {
		t.Fatalf("sessions = %+v", resp.Sessions)
	}
	if calls := f.all("info"); len(calls) != 0 {
		t.Fatalf("info jobs = %d, want 0 in read-only", len(calls))
	}
	if calls := f.all("windows"); len(calls) != 0 {
		t.Fatalf("windows jobs = %d, want 0 in read-only", len(calls))
	}
}

func TestSessionlessIDRoundTrip(t *testing.T) {
	id := sessionlessSessionID(216811801)
	if id != "sess-window-216811801" {
		t.Fatalf("id = %q", id)
	}
	if !isSessionlessSessionID(id) {
		t.Fatal("isSessionlessSessionID = false")
	}
	if got, ok := windowIDFromSessionlessID(id); !ok || got != 216811801 {
		t.Fatalf("windowIDFromSessionlessID = %d, %v", got, ok)
	}
	for _, bad := range []string{"sess-window-", "sess-window-abc", "sess-window-0", "sess-window--1", "sess-7lm3dk"} {
		if _, ok := windowIDFromSessionlessID(bad); ok {
			t.Fatalf("windowIDFromSessionlessID(%q) accepted", bad)
		}
	}
}

func TestAttachWindowValidatesRequestedID(t *testing.T) {
	m := &Manager{}
	if _, err := m.AttachWindow(context.Background(), 0, ""); err == nil {
		t.Fatal("window 0 accepted")
	}
	if _, err := m.AttachWindow(context.Background(), 950, "sess-window-951"); !errors.Is(err, errInvalidWindowSession) {
		t.Fatalf("mismatched id err = %v", err)
	}
}

func TestPickAllWindowsAnchorPrefersCapableLiveSession(t *testing.T) {
	rows := []sessionRow{
		{SessionID: "ai-critic", Extension: extensionInfo{Connected: true, Version: "1.0.20"}, SessionPageCount: 1},
		{SessionID: "sess-new", Extension: extensionInfo{Connected: true, Version: "1.0.21"}, SessionPageCount: 1},
	}
	if got := pickAllWindowsAnchor("ai-critic", rows, true); got != "sess-new" {
		t.Fatalf("anchor = %q, want sess-new", got)
	}
	// The current session still wins when it is itself capable.
	if got := pickAllWindowsAnchor("sess-new", rows, true); got != "sess-new" {
		t.Fatalf("current capable anchor = %q", got)
	}
}

func TestSessionsUsesCapableAnchorWhenCurrentIsStale(t *testing.T) {
	f := &fakeClient{rows: []sessionRow{
		{SessionID: "ai-critic", Extension: extensionInfo{Connected: true, Version: "1.0.20"}, SessionPageCount: 1},
		{SessionID: "sess-new", Extension: extensionInfo{Connected: true, Version: "1.0.21"}, SessionPageCount: 1},
	}}
	f.handle = func(call fakeCall) (json.RawMessage, bool) {
		switch call.Type {
		case "info":
			return json.RawMessage(`{"window_id":900,"tabs":[{"id":902,"index":0,"title":"Anchor","url":"https://anchor.example","active":true,"role":"user"}]}`), true
		case "windows":
			if call.Session != "sess-new" {
				return json.RawMessage(`{"features":["browser-agent"]}`), true
			}
			return json.RawMessage(`{"type":"windows","version":"1.0.21","features":["all-windows"],"windows":[` +
				`{"window_id":950,"title":"Design docs","tabs":[` +
				`{"tab_id":951,"index":0,"title":"Design docs","url":"https://design.example","active":true,"role":"user"}]}]}`), true
		}
		return nil, false
	}
	m := newTestManager(t, f)
	// The current adopted window is gone after a restart of the test Chrome.
	m.sessionID = "sess-window-999"
	m.connected = true
	m.windowID = 999
	m.contentTabID = 999

	resp := m.Sessions(context.Background())
	if !resp.OK || !resp.WindowsSupported {
		t.Fatalf("response = %+v, want windows_supported", resp)
	}
	call, ok := f.last("windows")
	if !ok || call.Session != "sess-new" {
		t.Fatalf("windows job session = %q, want sess-new", call.Session)
	}
	found := false
	for _, row := range resp.Sessions {
		if row.ID == "sess-window-950" && row.Sessionless {
			found = true
		}
	}
	if !found {
		t.Fatalf("sessions = %+v, want the discovered window", resp.Sessions)
	}
}
