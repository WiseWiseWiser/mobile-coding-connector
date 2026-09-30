package unified_tunnel

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIsHealthCheckHealthy(t *testing.T) {
	cases := []struct {
		code int
		want bool
	}{
		{200, true},
		{301, true},
		{401, true},
		{404, true},
		{499, true},
		{500, false},
		{502, false},
		{530, true},
		{0, false},
	}
	for _, tc := range cases {
		if got := IsHealthCheckHealthy(tc.code); got != tc.want {
			t.Errorf("IsHealthCheckHealthy(%d) = %v, want %v", tc.code, got, tc.want)
		}
	}
}

func TestIsEdgeRegistered(t *testing.T) {
	if IsEdgeRegistered(530) {
		t.Fatal("530 is not an edge registration")
	}
	if !IsEdgeRegistered(200) {
		t.Fatal("200 should count as registered")
	}
	if IsEdgeRegistered(502) {
		t.Fatal("502 is not registered")
	}
}

func TestProbeHostnameTreats530AsHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(530)
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(SetHealthCheckBaseURLForTest(srv.URL))

	utm, _ := testTunnelManager(t)
	if !utm.ProbeAndRecord("mac-agent-aes42.example.com") {
		t.Fatal("530 should be treated as healthy so health checks do not SIGTERM")
	}
	code, ok := utm.LastHealthProbe("mac-agent-aes42.example.com")
	if !ok || code != 530 {
		t.Fatalf("LastHealthProbe = %d ok=%v, want 530", code, ok)
	}
	if IsEdgeRegistered(code) {
		t.Fatal("status must not report active on 530")
	}
}

func TestProbeHostname200HealthyAndRegistered(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("pong"))
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(SetHealthCheckBaseURLForTest(srv.URL))

	utm, _ := testTunnelManager(t)
	if !utm.ProbeAndRecord("ok.example.com") {
		t.Fatal("200 should be healthy")
	}
	code, ok := utm.LastHealthProbe("ok.example.com")
	if !ok || code != 200 {
		t.Fatalf("LastHealthProbe = %d ok=%v, want 200", code, ok)
	}
	if !IsEdgeRegistered(code) {
		t.Fatal("200 should count as registered")
	}
}

func TestProbeHostname502Unhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(502)
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(SetHealthCheckBaseURLForTest(srv.URL))

	utm, _ := testTunnelManager(t)
	if utm.ProbeAndRecord("bad.example.com") {
		t.Fatal("502 should be unhealthy")
	}
}

func TestCloudflaredRunArgsPinsHTTP2(t *testing.T) {
	args := CloudflaredRunArgs("/tmp/cfg.yml", "my-tunnel", "")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--protocol http2") {
		t.Fatalf("args = %v, want --protocol http2", args)
	}
	if args[len(args)-1] != "my-tunnel" {
		t.Fatalf("tunnel ref should be last, got %v", args)
	}
}

func TestWithTunnelProtocolEnvReplacesExisting(t *testing.T) {
	env := withTunnelProtocolEnv([]string{"PATH=/bin", "TUNNEL_TRANSPORT_PROTOCOL=quic"}, "")
	var found string
	count := 0
	for _, e := range env {
		if strings.HasPrefix(e, "TUNNEL_TRANSPORT_PROTOCOL=") {
			count++
			found = e
		}
	}
	if count != 1 || found != "TUNNEL_TRANSPORT_PROTOCOL=http2" {
		t.Fatalf("env = %v", env)
	}
}

func TestGeneratedConfigPinsHTTP2Protocol(t *testing.T) {
	utm, _ := testTunnelManager(t)
	if err := utm.AddMapping(&IngressMapping{
		ID: "domain-mac", Hostname: "mac-agent-aes42.example.com", Service: "http://localhost:23712",
	}); err != nil {
		t.Fatalf("AddMapping: %v", err)
	}
	waitForRebuildCount(t, 1, time.Second)

	cfg := readGeneratedConfig(t, utm)
	if cfg.Protocol != DefaultTunnelProtocol {
		t.Fatalf("protocol = %q, want %q", cfg.Protocol, DefaultTunnelProtocol)
	}
}
