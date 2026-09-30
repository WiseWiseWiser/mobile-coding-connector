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

	if !checkDomainPing("mac-agent-aes42.example.com") {
		t.Fatal("Cloudflare 530 must not fail domain health checks")
	}
}

func TestCheckDomainPing502Unhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(502)
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(unified_tunnel.SetHealthCheckBaseURLForTest(srv.URL))

	if checkDomainPing("bad.example.com") {
		t.Fatal("502 should fail health checks")
	}
}
