package wsproxy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRemoteDirectPatterns(t *testing.T) {
	patterns, err := ParseRemoteDirectPatterns([]string{
		"git.example.com:22",
		"*.db.internal:6606",
		"exact.example.com",
	})
	if err != nil {
		t.Fatalf("ParseRemoteDirectPatterns: %v", err)
	}
	if len(patterns) != 3 {
		t.Fatalf("len=%d", len(patterns))
	}
	if patterns[0].Host != "git.example.com" || patterns[0].Port != 22 {
		t.Fatalf("patterns[0]=%+v", patterns[0])
	}
	if !patterns[1].Wildcard || patterns[1].Host != ".db.internal" || patterns[1].Port != 6606 {
		t.Fatalf("patterns[1]=%+v", patterns[1])
	}
}

func TestParseRemoteDirectPortOnlyRejected(t *testing.T) {
	_, err := ParseRemoteDirectPatterns([]string{":6606"})
	if err == nil || !strings.Contains(err.Error(), "port-only") {
		t.Fatalf("err=%v", err)
	}
}

func TestGenerateXrayConfigWithRemoteDirect(t *testing.T) {
	tmp := t.TempDir()
	SetTestConfigDir(tmp)
	defer SetTestConfigDir("")

	cfg := &Config{
		UpstreamProxy: "http://proxy.internal:3128",
		ListenPort:    41626,
		WSPath:        "/ws",
		UUID:          "00000000-0000-4000-8000-000000000001",
		RemoteDirect:  []string{"*.db.internal:6606", "db.example.com"},
	}
	if err := generateXrayConfig(cfg); err != nil {
		t.Fatalf("generateXrayConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, xrayDirName, xrayConfigFileName))
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	outs, _ := parsed["outbounds"].([]any)
	if len(outs) != 2 {
		t.Fatalf("outbounds=%d want 2: %s", len(outs), data)
	}
	direct, _ := outs[1].(map[string]any)
	if direct["protocol"] != "freedom" || direct["tag"] != xrayOutboundDirect {
		t.Fatalf("direct outbound=%v", direct)
	}
	routing, _ := parsed["routing"].(map[string]any)
	rules, _ := routing["rules"].([]any)
	if len(rules) != 3 {
		t.Fatalf("rules=%d want 3: %v", len(rules), rules)
	}
	r0, _ := rules[0].(map[string]any)
	domains, _ := r0["domain"].([]any)
	if len(domains) != 1 || domains[0] != "domain:db.internal" {
		t.Fatalf("rule0 domain=%v", domains)
	}
	if r0["port"] != "6606" || r0["outboundTag"] != xrayOutboundDirect {
		t.Fatalf("rule0=%v", r0)
	}
	r2, _ := rules[2].(map[string]any)
	if r2["outboundTag"] != xrayOutboundProxy {
		t.Fatalf("catch-all=%v", r2)
	}
}

func TestGenerateXrayConfigWithoutRemoteDirect(t *testing.T) {
	tmp := t.TempDir()
	SetTestConfigDir(tmp)
	defer SetTestConfigDir("")

	cfg := &Config{
		UpstreamProxy: "http://proxy.internal:3128",
		ListenPort:    41626,
		WSPath:        "/ws",
		UUID:          "00000000-0000-4000-8000-000000000001",
	}
	if err := generateXrayConfig(cfg); err != nil {
		t.Fatalf("generateXrayConfig: %v", err)
	}
	data, err := os.ReadFile(xrayConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	_ = json.Unmarshal(data, &parsed)
	if _, ok := parsed["routing"]; ok {
		t.Fatalf("routing should be omitted without remote_direct: %s", data)
	}
	outs, _ := parsed["outbounds"].([]any)
	if len(outs) != 1 {
		t.Fatalf("outbounds=%d want 1", len(outs))
	}
}
