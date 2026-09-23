package cloudflare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/xhd2015/ai-critic/server/cloudflareproxy"
)

// fakeProxyEdge stands in for the edge's control plane. It registers mappings
// and answers /dial with dialStatus; 0 upgrades the dial for real so a publish
// can become ready.
func fakeProxyEdge(t *testing.T, dialStatus int) *httptest.Server {
	t.Helper()
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/dial/"):
			if dialStatus != 0 {
				http.Error(w, "rejected", dialStatus)
				return
			}
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
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"mappings": []cloudflareproxy.MappingView{}})
		}
	}))
	t.Cleanup(ts.Close)
	return ts
}

func proxyCfg(edgeURL string) *CloudflareConfig {
	return &CloudflareConfig{Mode: "proxy", ProxyURL: edgeURL, Token: "secret"}
}

// TestProxySessionLifecycle covers both halves of the registry contract: a
// healthy publish registers its session, and stopping it removes the entry so
// status cannot keep reporting a session that is gone.
func TestProxySessionLifecycle(t *testing.T) {
	domain := "lifecycle.example.com"
	SetTestProxySession(domain, false)
	t.Cleanup(func() { SetTestProxySession(domain, false) })

	status, err := startViaProxy(domain, 1234, proxyCfg(fakeProxyEdge(t, 0).URL), func(string) {})
	if err != nil {
		t.Fatalf("startViaProxy() error = %v", err)
	}
	if status == nil || status.Status != "active" {
		t.Fatalf("status = %#v, want active", status)
	}
	if !proxySessionActive(domain) {
		t.Fatal("a ready publish did not register its session")
	}

	if !stopProxySession(domain) {
		t.Fatal("stopProxySession() reported nothing to stop")
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !proxySessionActive(domain) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("session stayed registered after the publish was stopped")
}

// TestDeadPublishClearsTheSessionRegistry is the regression test for the state
// that made the outage invisible and unrecoverable: a publish that ends on its
// own used to leave its session registered forever, so status claimed a session
// that no longer existed and no start path would republish it.
func TestDeadPublishClearsTheSessionRegistry(t *testing.T) {
	domain := "dead.example.com"
	SetTestProxySession(domain, false)
	t.Cleanup(func() { SetTestProxySession(domain, false) })

	// 401 is terminal for the pool, so the publish ends without being stopped.
	// startViaProxy itself fails (it never became ready), and either way the
	// registry must not keep an entry.
	_, _ = startViaProxy(domain, 1234, proxyCfg(fakeProxyEdge(t, http.StatusUnauthorized).URL), func(string) {})

	if proxySessionActive(domain) {
		t.Fatal("a publish that ended left its session registered; status would lie about it")
	}
}
