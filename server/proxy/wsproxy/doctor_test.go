package wsproxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDoctorReportsLocalXrayWithoutTunnel(t *testing.T) {
	const wsPath = "/ws"
	port, closeXray := startFakeXray(t, wsPath)
	defer closeXray()

	tmpDir := t.TempDir()
	cfg := &Config{
		UpstreamProxy: "http://proxy.internal:3128",
		ListenPort:    port,
		WSPath:        wsPath,
		UUID:          "00000000-0000-4000-8000-000000000001",
		Subdomain:     "ws",
		InstanceID:    "25b2a55939e4",
		PublicURL:     "https://ws-25b2a55939e4.xhd2015.xyz",
	}
	SetTestConfigDir(tmpDir)
	defer SetTestConfigDir("")
	if err := SaveTestConfig(cfg); err != nil {
		t.Fatalf("SaveTestConfig: %v", err)
	}

	m := NewTestManager(cfg.PublicURL, false)
	report := m.Doctor("https://example.com")

	if report.Healthy {
		t.Fatal("doctor should report unhealthy without tunnel mapping")
	}

	var clientReady, tunnelIngress DoctorCheckStatus
	for _, c := range report.Checks {
		switch c.ID {
		case "client_ready":
			clientReady = c.Status
		case "tunnel_ingress":
			tunnelIngress = c.Status
		}
	}
	if tunnelIngress != DoctorFail {
		t.Fatalf("tunnel_ingress status = %q, want fail", tunnelIngress)
	}
	if clientReady != DoctorFail {
		t.Fatalf("client_ready status = %q, want fail", clientReady)
	}
}

func TestDoctorDirectHTTPClientIgnoresProxyEnv(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Bad Request", http.StatusBadRequest)
	}))
	defer target.Close()

	t.Setenv("HTTP_PROXY", "http://127.0.0.1:9")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:9")

	req, err := http.NewRequest(http.MethodGet, target.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := doctorDirectHTTPClient(5 * time.Second).Do(req)
	if err != nil {
		t.Fatalf("direct client should bypass broken HTTP_PROXY: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestCheckUpstreamTCP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	check := checkUpstreamTCP(srv.URL)
	if check.Status != DoctorOK {
		t.Fatalf("status = %q, want ok (detail: %s)", check.Status, check.Detail)
	}
}