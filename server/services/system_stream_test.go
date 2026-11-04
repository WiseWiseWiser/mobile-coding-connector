package services

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xhd2015/agent-pro/agent/streaming/sse"
)

func sseWriterFor(w http.ResponseWriter) *sse.Writer {
	return sse.NewWriter(w)
}

// streamTestController implements only SystemController (status-only service).
type streamTestController struct{}

func (streamTestController) SystemStatus() SystemRuntime { return SystemRuntime{Detail: "idle"} }

// streamTestLifecycleController adds start/stop so the fallback path can run.
type streamTestLifecycleController struct{}

func (streamTestLifecycleController) SystemStatus() SystemRuntime {
	return SystemRuntime{Detail: "idle"}
}
func (streamTestLifecycleController) StartSystem() error { return nil }
func (streamTestLifecycleController) StopSystem() error  { return nil }

// streamTestStreamingController streams its own operation.
type streamTestStreamingController struct{}

func (streamTestStreamingController) SystemStatus() SystemRuntime {
	return SystemRuntime{Detail: "idle"}
}
func (streamTestStreamingController) StartSystem() error { return nil }
func (streamTestStreamingController) StopSystem() error  { return nil }
func (streamTestStreamingController) StreamSystemOperation(w http.ResponseWriter, action string) error {
	sw := sseWriterFor(w)
	if sw == nil {
		return fmt.Errorf("no flusher")
	}
	sw.SendLog("delegated " + action)
	sw.SendDone(map[string]string{"status": "ok", "action": action})
	return nil
}

func setupStreamServer(t *testing.T) *httptest.Server {
	t.Helper()
	m := NewManager()
	for _, svc := range []SystemService{
		{ID: "sys-stream", Name: "Streamer", Controller: streamTestStreamingController{}},
		{ID: "sys-plain", Name: "Plain", Controller: streamTestLifecycleController{}},
		{ID: "sys-status", Name: "StatusOnly", Controller: streamTestController{}},
	} {
		if err := m.RegisterSystemService(svc); err != nil {
			t.Fatalf("register %s: %v", svc.ID, err)
		}
	}
	mux := http.NewServeMux()
	RegisterAPIWithManager(mux, m)
	return httptest.NewServer(mux)
}

func TestSystemStreamDelegatesToStreamer(t *testing.T) {
	srv := setupStreamServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/services/system/stream?id=sys-stream&action=start")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	text := string(body)
	if !strings.Contains(text, `"message":"delegated start"`) {
		t.Fatalf("stream body missing delegated line: %s", text)
	}
	if !strings.Contains(text, `"type":"done"`) {
		t.Fatalf("stream body missing done: %s", text)
	}
}

func TestSystemStreamFallbackRunsAction(t *testing.T) {
	srv := setupStreamServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/services/system/stream?id=sys-plain&action=stop")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	text := string(body)
	if !strings.Contains(text, `"message":"stop Plain…"`) {
		t.Fatalf("fallback body missing action line: %s", text)
	}
	if !strings.Contains(text, `"status":"ok"`) {
		t.Fatalf("fallback body missing ok done: %s", text)
	}
}

func TestSystemStreamFallbackUnsupportedAction(t *testing.T) {
	srv := setupStreamServer(t)
	defer srv.Close()

	// Status-only controller: Start is unsupported → error path in-stream.
	resp, err := http.Get(srv.URL + "/api/services/system/stream?id=sys-status&action=start")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	text := string(body)
	if !strings.Contains(text, `"type":"error"`) {
		t.Fatalf("expected in-stream error, got: %s", text)
	}
}

func TestSystemStreamValidation(t *testing.T) {
	srv := setupStreamServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/services/system/stream?id=sys-stream&action=pause")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad action status = %d, want 400", resp.StatusCode)
	}

	resp2, err := http.Get(srv.URL + "/api/services/system/stream?id=sys-missing&action=start")
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Fatalf("missing id status = %d, want 404", resp2.StatusCode)
	}
}
