package domains

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xhd2015/ai-critic/server/cloudflare/unified_tunnel"
)

func TestCheckDomainPingTreats530AsHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(530)
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(unified_tunnel.SetHealthCheckBaseURLForTest(srv.URL))

	code, err := unified_tunnel.ProbeHostname("mac-agent-aes42.example.com")
	if err != nil || !unified_tunnel.IsHealthCheckHealthy(code) {
		t.Fatal("Cloudflare 530 must not fail domain health checks")
	}
}

func TestCheckDomainPing502Unhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(502)
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(unified_tunnel.SetHealthCheckBaseURLForTest(srv.URL))

	code, err := unified_tunnel.ProbeHostname("bad.example.com")
	if err != nil || unified_tunnel.IsHealthCheckHealthy(code) {
		t.Fatal("502 should fail health checks")
	}
}
