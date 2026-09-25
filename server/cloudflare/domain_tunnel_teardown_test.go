package cloudflare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/xhd2015/ai-critic/server/cloudflareproxy"
)

// recordingEdge is a fake cloudflare-proxy edge that records mapping deletes, so
// a test can assert the hostname was unpublished rather than merely unregistered
// locally. A locally-cleared session with a surviving edge mapping is exactly
// the state that answered every visitor with 502.
type recordingEdge struct {
	*httptest.Server
	mu      sync.Mutex
	deleted []string
}

func newRecordingEdge(t *testing.T) *recordingEdge {
	t.Helper()
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	edge := &recordingEdge{}
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/dial/"):
			conn, err := up.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		case r.Method == http.MethodPost && r.URL.Path == "/api/mappings":
			_ = json.NewEncoder(w).Encode(cloudflareproxy.MappingView{
				ID:       "map-1",
				Hostname: "app.example.com",
				DialURL:  "ws" + strings.TrimPrefix(ts.URL, "http") + "/dial/map-1?t=tok",
			})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/mappings":
			id := r.URL.Query().Get("id")
			edge.mu.Lock()
			edge.deleted = append(edge.deleted, id)
			edge.mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]string{"deleted": id})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"mappings": []cloudflareproxy.MappingView{}})
		}
	}))
	t.Cleanup(ts.Close)
	edge.Server = ts
	return edge
}

func (e *recordingEdge) deletedIDs() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.deleted...)
}

// TestStopDomainTunnelDeletesEdgeMapping asserts the full teardown: stopping a
// proxy-mode publish must delete its edge mapping, not just clear the local
// session. Without the delete the edge keeps serving a mapping whose dial pool
// is gone, which is the 502 this fixes.
func TestStopDomainTunnelDeletesEdgeMapping(t *testing.T) {
	const domain = "teardown.example.com"
	SetTestProxySession(domain, false)
	t.Cleanup(func() { SetTestProxySession(domain, false) })

	edge := newRecordingEdge(t)
	status, err := startViaProxy(domain, 1234, proxyCfg(edge.URL), func(string) {})
	if err != nil {
		t.Fatalf("startViaProxy() error = %v", err)
	}
	if status == nil || status.Status != "active" {
		t.Fatalf("status = %#v, want active", status)
	}

	if err := StopDomainTunnel(domain, ""); err != nil {
		t.Fatalf("StopDomainTunnel() error = %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(edge.deletedIDs()) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	deleted := edge.deletedIDs()
	if len(deleted) == 0 {
		t.Fatal("stopping the publish left its mapping on the edge")
	}
	if deleted[0] != "map-1" {
		t.Fatalf("deleted %q, want %q", deleted[0], "map-1")
	}
}
