package main

import (
	"strings"
	"testing"
)

func TestBuildDoctorXrayClientConfig(t *testing.T) {
	cfg := buildDoctorXrayClientConfig(&doctorVMess{
		Host:    "ws.example.com",
		Port:    "443",
		UUID:    "uuid-1",
		Network: "ws",
		Path:    "/ws",
		TLS:     "tls",
	}, 18080)

	if !strings.Contains(cfg, `"address": "ws.example.com"`) {
		t.Fatalf("missing vmess address: %s", cfg)
	}
	if !strings.Contains(cfg, `"port": 18080`) {
		t.Fatalf("missing inbound port: %s", cfg)
	}
	if !strings.Contains(cfg, `"path": "/ws"`) {
		t.Fatalf("missing ws path: %s", cfg)
	}
}

func TestDoctorStatusTag(t *testing.T) {
	if doctorStatusTag("ok") != "[ok]  " {
		t.Fatalf("unexpected ok tag: %q", doctorStatusTag("ok"))
	}
	if doctorStatusTag("fail") != "[fail]" {
		t.Fatalf("unexpected fail tag: %q", doctorStatusTag("fail"))
	}
}