package domains

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cloudflareSettings "github.com/xhd2015/ai-critic/server/cloudflare"
)

func TestEnsurePersistedTunnelNameGenerates(t *testing.T) {
	dir := t.TempDir()
	orig := getDomainsFile()
	SetDomainsFile(filepath.Join(dir, "server-domains.json"))
	t.Cleanup(func() { SetDomainsFile(orig) })

	name, err := EnsurePersistedTunnelName()
	if err != nil {
		t.Fatal(err)
	}
	wantPrefix := "ai-critic-"
	if !strings.HasPrefix(name, wantPrefix) {
		t.Fatalf("name %q", name)
	}
	again, err := EnsurePersistedTunnelName()
	if err != nil {
		t.Fatal(err)
	}
	if again != name {
		t.Fatalf("expected persist %q got %q", name, again)
	}
	cfg, err := LoadDomains()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TunnelName != name {
		t.Fatalf("file %q", cfg.TunnelName)
	}
}

func TestEnsurePersistedTunnelNameKeepsExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server-domains.json")
	orig := getDomainsFile()
	SetDomainsFile(path)
	t.Cleanup(func() { SetDomainsFile(orig) })
	if err := os.WriteFile(path, []byte(`{"domains":[],"tunnel_name":"ai-critic-custom"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	name, err := EnsurePersistedTunnelName()
	if err != nil {
		t.Fatal(err)
	}
	if name != "ai-critic-custom" {
		t.Fatalf("got %q", name)
	}
}

func TestSanitizeMatchesHelper(t *testing.T) {
	h := cloudflareSettings.HostPreferredTunnelName("My.Host")
	if h != "ai-critic-my" {
		t.Fatalf("got %q", h)
	}
}
