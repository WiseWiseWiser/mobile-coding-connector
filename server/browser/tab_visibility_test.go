package browser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestCreatedContentTabIsInInitialSnapshot(t *testing.T) {
	f := healthyFake()
	f.info = `{"window_id":900,"tabs":[{"id":901,"index":0,"url":"http://127.0.0.1:43799/go?session=ai-critic","active":true,"role":"session_page"}]}`
	f.handle = func(call fakeCall) (json.RawMessage, bool) {
		if call.Type == "create_tab" {
			return json.RawMessage(`{"tab_id":902}`), true
		}
		return nil, false
	}
	m := newTestManager(t, f)
	if err := m.ensureWindow(context.Background()); err != nil {
		t.Fatal(err)
	}
	status := m.Snapshot()
	if status.ActiveTabID != 902 || len(status.Tabs) != 2 {
		t.Fatalf("initial snapshot missing newly created tab: %+v", status)
	}
	tab := status.Tabs[1]
	if tab.TabID != 902 || tab.Role != "user" || tab.Title != "ai-critic" || !tab.Active || tab.URL != contentPlaceholderURL() {
		t.Fatalf("new tab: %+v", tab)
	}
	if status.Tabs[0].Active {
		t.Fatal("session page remained active")
	}
}

func TestTabPollingBroadcastsChangedList(t *testing.T) {
	f := healthyFake()
	f.info = `{"window_id":900,"tabs":[{"id":901,"url":"http://127.0.0.1:43799/go?session=ai-critic","role":"session_page"},{"id":902,"url":"data:text/html,content","title":"ai-critic","active":true,"role":"user"}]}`
	m := newTestManager(t, f)
	m.sessionID, m.connected, m.windowID, m.contentTabID = "ai-critic", true, 900, 902
	m.tabs = []TabInfo{{TabID: 901, Role: "session_page"}}
	registered := make(chan *viewer, 1)
	done := make(chan struct{})
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		v := &viewer{conn: conn}
		m.hub.mu.Lock()
		m.hub.viewers[v] = struct{}{}
		m.hub.mu.Unlock()
		registered <- v
		<-done
		m.hub.mu.Lock()
		delete(m.hub.viewers, v)
		m.hub.mu.Unlock()
		_ = conn.Close()
	}))
	defer srv.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	<-registered
	defer close(done)
	m.pollTabs(context.Background())
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var msg struct {
		Type   string `json:"type"`
		Status Status `json:"status"`
	}
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("missing status broadcast: %v", err)
	}
	if msg.Type != "status" || len(msg.Status.Tabs) != 2 || msg.Status.Tabs[1].Title != "ai-critic" {
		t.Fatalf("status: %+v", msg)
	}
}
