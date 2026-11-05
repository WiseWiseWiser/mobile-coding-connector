package wsproxy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateXrayConfigEnablesSniffing(t *testing.T) {
	dir := t.TempDir()
	SetTestConfigDir(dir)
	defer SetTestConfigDir("")

	cfg := &Config{
		UpstreamProxy: "http://127.0.0.1:23456",
		ListenPort:    41626,
		WSPath:        "/ws",
		UUID:          "00000000-0000-4000-8000-000000000001",
	}
	if err := generateXrayConfig(cfg); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, xrayDirName, xrayConfigFileName))
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Inbounds []struct {
			Sniffing *xraySniffing `json:"sniffing"`
		} `json:"inbounds"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Inbounds) != 1 || parsed.Inbounds[0].Sniffing == nil {
		t.Fatalf("missing sniffing: %s", data)
	}
	sn := parsed.Inbounds[0].Sniffing
	if !sn.Enabled {
		t.Fatal("sniffing.enabled = false")
	}
	want := map[string]bool{"http": true, "tls": true}
	for _, d := range sn.DestOverride {
		delete(want, d)
	}
	if len(want) > 0 {
		t.Fatalf("destOverride missing %v in %v", want, sn.DestOverride)
	}
}
