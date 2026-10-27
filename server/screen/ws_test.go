package screen

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/xhd2015/ai-critic/server/config"
)

// testServer wires RegisterAPI to a manager whose capture is disabled and whose
// backend is fake, so the HTTP and WebSocket layers can be exercised without
// touching the host screen.
func testServer(t *testing.T) (*httptest.Server, *Manager, *fakeBackend) {
	t.Helper()
	m := Default()
	be := &fakeBackend{}
	m.stop()

	m.mu.Lock()
	m.backend = be
	m.cfg.Enabled = false
	m.cfg.AllowControl = true
	m.mu.Unlock()

	t.Cleanup(func() {
		m.mu.Lock()
		m.backend = newBackend()
		m.cfg.Enabled = true
		m.mu.Unlock()
		for _, v := range m.viewersSnapshot() {
			m.removeViewer(v)
		}
	})

	mux := http.NewServeMux()
	RegisterAPI(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, m, be
}

func (m *Manager) viewersSnapshot() []*viewer {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*viewer, 0, len(m.viewers))
	for v := range m.viewers {
		out = append(out, v)
	}
	return out
}

func dial(t *testing.T, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/screen/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		if resp != nil {
			t.Fatalf("dial: %v (status %d)", err, resp.StatusCode)
		}
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// readJSON reads the next text message and decodes it into a generic map.
func readJSON(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	kind, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if kind != websocket.TextMessage {
		t.Fatalf("message kind = %d, want text", kind)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("json: %v (%q)", err, data)
	}
	return out
}

func waitViewers(t *testing.T, m *Manager, want int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if m.Snapshot().Viewers == want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("viewers = %d, want %d", m.Snapshot().Viewers, want)
}

func TestWebSocketSendsHelloAndStatus(t *testing.T) {
	srv, m, _ := testServer(t)

	conn := dial(t, srv)
	hello := readJSON(t, conn)
	if hello["type"] != "hello" {
		t.Fatalf("first message = %v", hello)
	}
	if id, _ := hello["viewer_id"].(string); id == "" {
		t.Fatalf("hello has no viewer_id: %v", hello)
	}
	if _, ok := hello["status"].(map[string]any); !ok {
		t.Fatalf("hello has no status: %v", hello)
	}
	waitViewers(t, m, 1)
}

func TestWebSocketControlAndInputRoundTrip(t *testing.T) {
	srv, m, be := testServer(t)

	conn := dial(t, srv)
	readJSON(t, conn) // hello

	if err := conn.WriteJSON(map[string]any{"type": "request_control"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	reply := readJSON(t, conn)
	for i := 0; reply["type"] != "control" && i < 5; i++ {
		reply = readJSON(t, conn)
	}
	if reply["type"] != "control" || reply["granted"] != true {
		t.Fatalf("control reply = %v", reply)
	}

	// Keep the frame geometry usable without a capture pipeline.
	m.mu.Lock()
	m.displays = []Display{{ID: "main", W: 1728, H: 1117, Scale: 2}}
	m.imgW, m.imgH = 1280, 828
	m.mu.Unlock()

	if err := conn.WriteMessage(websocket.BinaryMessage, pointerMsg(actionMove, buttonNone, 640, 414, 0)); err != nil {
		t.Fatalf("write pointer: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && len(be.snapshot()) == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	calls := be.snapshot()
	if len(calls) == 0 {
		t.Fatal("pointer message did not reach the backend")
	}
	if got := calls[0]; got.name != "move" || got.a != 864 || got.b != 558 {
		t.Fatalf("backend call = %+v, want move 864,558", got)
	}

	// Releasing control must lift every held button.
	if err := conn.WriteJSON(map[string]any{"type": "release_control"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if m.Snapshot().Control.Holder == "" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("control was not released")
}

func TestWebSocketSecondViewerCannotControl(t *testing.T) {
	srv, m, _ := testServer(t)

	first := dial(t, srv)
	readJSON(t, first)
	if err := first.WriteJSON(map[string]any{"type": "request_control"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	for i := 0; i < 5; i++ {
		if msg := readJSON(t, first); msg["type"] == "control" {
			break
		}
	}
	waitViewers(t, m, 1)

	second := dial(t, srv)
	readJSON(t, second) // hello
	if err := second.WriteJSON(map[string]any{"type": "request_control"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	reply := readJSON(t, second)
	for i := 0; reply["type"] != "control" && i < 5; i++ {
		reply = readJSON(t, second)
	}
	if reply["granted"] != false {
		t.Fatalf("second viewer was granted control: %v", reply)
	}
	if reason, _ := reply["reason"].(string); reason == "" {
		t.Fatalf("denial without reason: %v", reply)
	}

	// Input from the second viewer is dropped and answered with an error.
	if err := second.WriteMessage(websocket.BinaryMessage, pointerMsg(actionMove, buttonNone, 10, 10, 0)); err != nil {
		t.Fatalf("write pointer: %v", err)
	}
	msg := readJSON(t, second)
	for i := 0; msg["type"] != "error" && i < 5; i++ {
		msg = readJSON(t, second)
	}
	if msg["code"] != "not_controller" {
		t.Fatalf("second viewer got %v, want not_controller", msg)
	}
}

// The control messages must name the released viewer and the remaining holder:
// a client uses them to tell "I released" (nobody controls) apart from "someone
// else is driving".
func TestControlMessagesReportHolder(t *testing.T) {
	srv, _, _ := testServer(t)

	first := dial(t, srv)
	hello := readJSON(t, first)
	firstID, _ := hello["viewer_id"].(string)
	if firstID == "" {
		t.Fatalf("hello without viewer id: %v", hello)
	}
	if err := first.WriteJSON(map[string]any{"type": "request_control"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	if msg := readControl(t, first); msg["granted"] != true {
		t.Fatalf("first viewer was denied control: %v", msg)
	}

	second := dial(t, srv)
	readJSON(t, second)
	if err := second.WriteJSON(map[string]any{"type": "request_control"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	denial := readControl(t, second)
	if denial["granted"] != false {
		t.Fatalf("second viewer was granted control: %v", denial)
	}
	if denial["holder"] != firstID {
		t.Fatalf("denial holder = %v, want %v", denial["holder"], firstID)
	}

	if err := first.WriteJSON(map[string]any{"type": "release_control"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	for name, conn := range map[string]*websocket.Conn{"released": first, "bystander": second} {
		released := readControl(t, conn)
		if released["granted"] != false {
			t.Fatalf("%s: release reported granted: %v", name, released)
		}
		if released["holder"] != "" {
			t.Fatalf("%s: holder after release = %v, want empty", name, released["holder"])
		}
		if released["viewer_id"] != firstID {
			t.Fatalf("%s: released viewer = %v, want %v", name, released["viewer_id"], firstID)
		}
	}
}

func readControl(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	for i := 0; i < 10; i++ {
		if msg := readJSON(t, conn); msg["type"] == "control" {
			return msg
		}
	}
	t.Fatalf("no control message")
	return nil
}

func TestWebSocketStatusRequest(t *testing.T) {
	srv, _, _ := testServer(t)

	conn := dial(t, srv)
	readJSON(t, conn) // hello
	if err := conn.WriteJSON(map[string]any{"type": "status"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	msg := readJSON(t, conn)
	for i := 0; msg["type"] != "status" && i < 5; i++ {
		msg = readJSON(t, conn)
	}
	if msg["type"] != "status" {
		t.Fatalf("status reply = %v", msg)
	}
	if _, ok := msg["status"].(map[string]any); !ok {
		t.Fatalf("status payload = %v", msg["status"])
	}
}

func TestWebSocketDisconnectRemovesViewer(t *testing.T) {
	srv, m, _ := testServer(t)

	conn := dial(t, srv)
	readJSON(t, conn)
	waitViewers(t, m, 1)

	if err := conn.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	waitViewers(t, m, 0)
}

func TestStatusEndpointJSON(t *testing.T) {
	srv, _, _ := testServer(t)

	resp, err := http.Get(srv.URL + "/api/screen/status")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var st Status
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if st.Platform == "" {
		t.Fatal("status has no platform")
	}
	if st.Control.Enabled != true {
		t.Fatalf("control enabled = %v", st.Control.Enabled)
	}
}

// preserveConfigFile restores the on-disk screen config (and removes the data
// directory when the test created it), so tests never overwrite real settings.
func preserveConfigFile(t *testing.T) {
	t.Helper()
	path := config.ScreenConfFile
	prior, err := os.ReadFile(path)
	hadFile := err == nil
	dir := filepath.Dir(path)
	_, dirErr := os.Stat(dir)
	hadDir := dirErr == nil

	t.Cleanup(func() {
		if hadFile {
			_ = os.WriteFile(path, prior, 0644)
			return
		}
		_ = os.Remove(path)
		if !hadDir {
			_ = os.Remove(dir)
		}
	})
}

func TestConfigEndpointNormalizes(t *testing.T) {
	srv, m, _ := testServer(t)
	preserveConfigFile(t)
	t.Cleanup(func() {
		m.mu.Lock()
		m.cfg = DefaultConfig()
		m.mu.Unlock()
	})

	body := `{"enabled":false,"allow_control":false,"max_width":100,"fps":999,"idle_fps":900,"quality":100,"max_viewers":0}`
	resp, err := http.Post(srv.URL+"/api/screen/config", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var cfg Config
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.MaxWidth < 320 || cfg.MaxWidth > 2560 {
		t.Fatalf("max_width = %d", cfg.MaxWidth)
	}
	if cfg.FPS > 30 || cfg.FPS < 1 {
		t.Fatalf("fps = %d", cfg.FPS)
	}
	if cfg.IdleFPS > cfg.FPS {
		t.Fatalf("idle_fps = %d > fps = %d", cfg.IdleFPS, cfg.FPS)
	}
	if cfg.Quality > 90 {
		t.Fatalf("quality = %d", cfg.Quality)
	}
	if cfg.MaxViewers < 1 {
		t.Fatalf("max_viewers = %d", cfg.MaxViewers)
	}
	if m.Config().AllowControl {
		t.Fatal("allow_control must be false after update")
	}
}

func TestConfigEndpointKeepsOmittedFields(t *testing.T) {
	srv, m, _ := testServer(t)
	preserveConfigFile(t)
	t.Cleanup(func() {
		m.mu.Lock()
		m.cfg = DefaultConfig()
		m.mu.Unlock()
	})

	m.mu.Lock()
	m.cfg = Config{Enabled: true, AllowControl: true, MaxWidth: 1280, FPS: 12, IdleFPS: 3, Quality: 55, MaxViewers: 4}
	m.mu.Unlock()

	resp, err := http.Post(srv.URL+"/api/screen/config", "application/json", strings.NewReader(`{"fps":20}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var cfg Config
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !cfg.Enabled || !cfg.AllowControl {
		t.Fatalf("partial update cleared flags: %+v", cfg)
	}
	if cfg.FPS != 20 {
		t.Fatalf("fps = %d, want 20", cfg.FPS)
	}
	if cfg.MaxWidth != 1280 || cfg.Quality != 55 {
		t.Fatalf("partial update changed untouched fields: %+v", cfg)
	}
}

func TestConfigEndpointRejectsBadJSON(t *testing.T) {
	srv, _, _ := testServer(t)

	resp, err := http.Post(srv.URL+"/api/screen/config", "application/json", strings.NewReader("{"))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestReleaseControlEndpoint(t *testing.T) {
	srv, m, _ := testServer(t)

	conn := dial(t, srv)
	readJSON(t, conn)
	if err := conn.WriteJSON(map[string]any{"type": "request_control"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	for i := 0; i < 5; i++ {
		if msg := readJSON(t, conn); msg["type"] == "control" {
			break
		}
	}
	if m.Snapshot().Control.Holder == "" {
		t.Fatal("control not granted")
	}

	resp, err := http.Post(srv.URL+"/api/screen/control/release", "application/json", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if m.Snapshot().Control.Holder != "" {
		t.Fatal("control still held after admin release")
	}
}
