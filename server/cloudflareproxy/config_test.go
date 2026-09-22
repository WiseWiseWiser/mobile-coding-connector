package cloudflareproxy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareAuthRotatesWhenTokenEmpty(t *testing.T) {
	cfg := Config{GeneratedToken: "old"}
	rotated, err := prepareAuth(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !rotated {
		t.Fatal("expected rotation")
	}
	if cfg.GeneratedToken == "" || cfg.GeneratedToken == "old" {
		t.Fatalf("generatedToken not rotated: %q", cfg.GeneratedToken)
	}
	if cfg.Token != "" {
		t.Fatalf("token should stay empty, got %q", cfg.Token)
	}
}

func TestPrepareAuthClearsGeneratedWhenTokenSet(t *testing.T) {
	cfg := Config{Token: "fixed", GeneratedToken: "old"}
	rotated, err := prepareAuth(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	if rotated {
		t.Fatal("configured token should not rotate")
	}
	if cfg.Token != "fixed" {
		t.Fatalf("token mutated: %q", cfg.Token)
	}
	if cfg.GeneratedToken != "" {
		t.Fatalf("generatedToken should be cleared, got %q", cfg.GeneratedToken)
	}
}

func TestSaveLoadConfigRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := Config{Token: "abc", Domain: "cf-proxy.example.com", AutoStart: true}
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Token != "abc" || got.Domain != "cf-proxy.example.com" || !got.AutoStart {
		t.Fatalf("got %+v", got)
	}
}

func TestLoadClientFileIgnoresOwnedDomains(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cloudflare.json")
	if err := os.WriteFile(path, []byte(`{"owned_domains":["xhd2015.xyz"],"proxy_url":"https://cf.example","token":"t1"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cf, err := LoadClientFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cf.ProxyURL != "https://cf.example" || cf.Token != "t1" {
		t.Fatalf("got %+v", cf)
	}
}
