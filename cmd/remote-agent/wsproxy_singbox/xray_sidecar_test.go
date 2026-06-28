package wsproxy_singbox

import (
	"strings"
	"testing"
)

func TestBuildXrayVMessClientConfig(t *testing.T) {
	cfg := BuildXrayVMessClientConfig(&VMessParams{
		Host:    "ws.example.com",
		Port:    "443",
		UUID:    "11111111-2222-4333-8444-555555555555",
		Network: "ws",
		Path:    "/ws",
		TLS:     "tls",
	}, 11080)
	if !strings.Contains(cfg, `"protocol": "http"`) {
		t.Fatalf("missing http inbound for doctor: %s", cfg)
	}
	if strings.Contains(cfg, `"disableFallback"`) {
		t.Fatalf("doctor config should not include sidecar DNS: %s", cfg)
	}
	if !strings.Contains(cfg, `"port": 11080`) {
		t.Fatalf("missing proxy port: %s", cfg)
	}
	if !strings.Contains(cfg, `"address": "ws.example.com"`) {
		t.Fatalf("missing vmess hostname: %s", cfg)
	}
	if strings.Contains(cfg, "sockopt") {
		t.Fatalf("doctor path should not use sockopt: %s", cfg)
	}
}

func TestBuildXraySidecarConfigUsesSOCKS(t *testing.T) {
	cfg := buildXraySidecarConfig(&VMessParams{
		Host: "ws.example.com", Port: "443", UUID: "u", Path: "/ws", TLS: "tls",
	}, 11081)
	if !strings.Contains(cfg, `"protocol": "socks"`) {
		t.Fatalf("missing socks inbound: %s", cfg)
	}
	if !strings.Contains(cfg, `"disableFallback": true`) {
		t.Fatalf("missing disableFallback DNS: %s", cfg)
	}
}

func TestBuildSingBoxTunConfigSocksOutbound(t *testing.T) {
	vmess := &VMessParams{
		Host: "proxy.example.com",
		Port: "443",
		UUID: "11111111-2222-4333-8444-555555555555",
		Path: "/ws",
		TLS:  "tls",
	}
	data, err := BuildSingBoxTunConfig(vmess, &BuildConfigOptions{LocalSocksPort: 11080})
	if err != nil {
		t.Fatalf("BuildSingBoxTunConfig: %v", err)
	}
	if !strings.Contains(string(data), `"type": "socks"`) {
		t.Fatalf("expected socks outbound: %s", data)
	}
	if strings.Contains(string(data), `"type": "vmess"`) {
		t.Fatalf("vmess outbound should be replaced by socks: %s", data)
	}
}