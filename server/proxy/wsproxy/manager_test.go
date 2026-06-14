package wsproxy

import (
	"encoding/base64"
	"encoding/json"
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
	m := &Manager{
		publicURL: "https://ws.example.com",
	}

	cfg := &Config{
		UUID:   "test-uuid-1234",
		WSPath: "/ws",
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
	m := &Manager{
		publicURL: "https://ws.example.com",
	}

	cfg := &Config{
		UUID:   "test-uuid-1234",
		WSPath: "/ws",
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
