package wsproxy

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

)

func TestGenerateUUID(t *testing.T) {
	uuid := generateUUID()
	parts := strings.Split(uuid, "-")
	if len(parts) != 5 {
		t.Errorf("UUID should have 5 parts, got %d: %s", len(parts), uuid)
	}
	if len(uuid) != 36 {
		t.Errorf("UUID should be 36 chars, got %d: %s", len(uuid), uuid)
	}

	uuid2 := generateUUID()
	if uuid == uuid2 {
		t.Error("two UUIDs should be different")
	}
}

func TestExtractHost(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"http://proxy.example.com:3128", "proxy.example.com"},
		{"http://proxy.example.com", "proxy.example.com"},
		{"https://proxy.example.com:8080", "proxy.example.com"},
		{"proxy.example.com:3128", "proxy.example.com"},
		{"proxy.example.com", "proxy.example.com"},
	}

	for _, tt := range tests {
		got := extractHost(tt.input)
		if got != tt.expected {
			t.Errorf("extractHost(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestExtractPort(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"http://proxy.example.com:3128", 3128},
		{"http://proxy.example.com:8080", 8080},
		{"http://proxy.example.com", 3128},
		{"proxy.example.com:9999", 9999},
	}

	for _, tt := range tests {
		got := extractPort(tt.input)
		if got != tt.expected {
			t.Errorf("extractPort(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestVMessLinkEncoding(t *testing.T) {
	const wsPath = "/ws"
	port, closeXray := startFakeXray(t, wsPath)
	defer closeXray()
	seedTunnelMapping("ws.example.com", port)

	m := &Manager{
		publicURL: "https://ws.example.com",
	}

	cfg := &Config{
		UUID:       "test-uuid-1234",
		WSPath:     wsPath,
		ListenPort: port,
		PublicURL:  "https://ws.example.com",
	}

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "ws-proxy.json")
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(cfgPath, append(data, '\n'), 0644)

	_testConfigDir = tmpDir
	defer func() { _testConfigDir = "" }()

	link := m.GetVMessLink()

	if !strings.HasPrefix(link, "vmess://") {
		t.Fatal("vmess link should start with vmess://")
	}

	encoded := strings.TrimPrefix(link, "vmess://")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("failed to decode vmess link: %v", err)
	}

	var vmess map[string]interface{}
	if err := json.Unmarshal(decoded, &vmess); err != nil {
		t.Fatalf("failed to parse vmess JSON: %v (raw: %s)", err, string(decoded))
	}

	if vmess["add"] != "ws.example.com" {
		t.Errorf("add: got %v, want ws.example.com", vmess["add"])
	}
	if vmess["port"] != "443" {
		t.Errorf("port: got %v, want 443", vmess["port"])
	}
	if vmess["net"] != "ws" {
		t.Errorf("net: got %v, want ws", vmess["net"])
	}
	if vmess["tls"] != "tls" {
		t.Errorf("tls: got %v, want tls", vmess["tls"])
	}
	if vmess["path"] != "/ws" {
		t.Errorf("path: got %v, want /ws", vmess["path"])
	}
}

func TestGetVMessConfig(t *testing.T) {
	const wsPath = "/ws"
	port, closeXray := startFakeXray(t, wsPath)
	defer closeXray()
	seedTunnelMapping("ws.example.com", port)

	m := &Manager{
		publicURL: "https://ws.example.com",
	}

	cfg := &Config{
		UUID:       "test-uuid-1234",
		WSPath:     wsPath,
		ListenPort: port,
		PublicURL:  "https://ws.example.com",
	}

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "ws-proxy.json")
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(cfgPath, append(data, '\n'), 0644)

	_testConfigDir = tmpDir
	defer func() { _testConfigDir = "" }()

	vmessCfg, err := m.GetVMessConfig()
	if err != nil {
		t.Fatalf("GetVMessConfig failed: %v", err)
	}

	if vmessCfg.Host != "ws.example.com" {
		t.Errorf("Host: got %q, want ws.example.com", vmessCfg.Host)
	}
	if vmessCfg.Port != "443" {
		t.Errorf("Port: got %q, want 443", vmessCfg.Port)
	}
	if vmessCfg.UUID != "test-uuid-1234" {
		t.Errorf("UUID: got %q, want test-uuid-1234", vmessCfg.UUID)
	}
	if vmessCfg.AlterID != "0" {
		t.Errorf("AlterID: got %q, want 0", vmessCfg.AlterID)
	}
	if vmessCfg.Network != "ws" {
		t.Errorf("Network: got %q, want ws", vmessCfg.Network)
	}
	if vmessCfg.Type != "none" {
		t.Errorf("Type: got %q, want none", vmessCfg.Type)
	}
	if vmessCfg.Path != "/ws" {
		t.Errorf("Path: got %q, want /ws", vmessCfg.Path)
	}
	if vmessCfg.TLS != "tls" {
		t.Errorf("TLS: got %q, want tls", vmessCfg.TLS)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg.ListenPort != 0 {
		t.Errorf("default port: got %d, want 0 (auto-assign)", cfg.ListenPort)
	}
	if cfg.WSPath != "/ws" {
		t.Errorf("default ws path: got %q, want /ws", cfg.WSPath)
	}
	if cfg.Subdomain != "ws" {
		t.Errorf("default subdomain: got %q, want ws", cfg.Subdomain)
	}
}

func TestStatusDefaults(t *testing.T) {
	m := &Manager{}
	status := m.Status()

	if status.Running {
		t.Error("new manager should not be running")
	}
	if status.Port == 0 {
		t.Error("status port should be non-zero (auto-assigned)")
	}
	if status.PublicURL != "" {
		t.Error("new manager should have empty URL")
	}
}

func seedTunnelMapping(hostname string, port int) {
	SetTestTunnelMapped(hostname, port, true)
}

func startFakeXray(t *testing.T, wsPath string) (port int, cleanup func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == wsPath {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		http.NotFound(w, r)
	}))
	port = ExtractPortFromURL(srv.URL)
	return port, srv.Close
}

func TestStatusWithoutTunnelMappingMustNotClaimRunning(t *testing.T) {
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
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(filepath.Join(tmpDir, configFileName), append(data, '\n'), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	SetTestConfigDir(tmpDir)
	defer SetTestConfigDir("")

	publicURL := "https://ws-25b2a55939e4.xhd2015.xyz"
	m := NewTestManager(publicURL, false)

	if !IsXrayAliveForTest(port, wsPath) {
		t.Fatal("precondition: simulated xray must be alive")
	}

	status := m.Status()
	link := m.GetVMessLink()
	if link != "" {
		t.Fatal("precondition: vmess link should be withheld without tunnel mapping")
	}

	if status.Running {
		t.Fatalf("BUG: Status.Running=true with publicURL=%q but no Cloudflare ingress; clients get ERR_PROXY_CONNECTION_FAILED (public /ws returns 404)", status.PublicURL)
	}
	if IsClientReady(status, false, link) {
		t.Fatalf("client-ready must be false without tunnel mapping; status=%+v link=%q", status, link)
	}
}

func TestStatusReconstructsPublicURLAfterRestart(t *testing.T) {
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
		AutoStart:     true,
		PublicURL:     "https://ws-25b2a55939e4.xhd2015.xyz",
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(filepath.Join(tmpDir, configFileName), append(data, '\n'), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	SetTestConfigDir(tmpDir)
	defer SetTestConfigDir("")

	m := NewTestManager("", false)
	status := m.Status()
	wantURL := cfg.PublicURL

	if status.PublicURL != wantURL {
		t.Fatalf("Status.PublicURL = %q, want %q (restored from persisted config after restart)", status.PublicURL, wantURL)
	}
	if status.Running {
		t.Fatal("running must be false without tunnel ingress even when public URL is restored")
	}
	if m.GetVMessLink() != "" {
		t.Fatal("vmess link must be withheld without tunnel ingress")
	}
}

func TestVMessLinkWithheldWithoutTunnelIngress(t *testing.T) {
	const wsPath = "/ws"
	port, closeXray := startFakeXray(t, wsPath)
	defer closeXray()

	tmpDir := t.TempDir()
	cfg := &Config{
		UpstreamProxy: "http://proxy.internal:3128",
		ListenPort:    port,
		WSPath:        wsPath,
		UUID:          "00000000-0000-4000-8000-000000000001",
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(filepath.Join(tmpDir, configFileName), append(data, '\n'), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	SetTestConfigDir(tmpDir)
	defer SetTestConfigDir("")

	m := NewTestManager("https://ws-25b2a55939e4.xhd2015.xyz", false)
	link := m.GetVMessLink()
	if link != "" {
		t.Fatalf("vmess link must be withheld without tunnel ingress; got %q", link)
	}
}
