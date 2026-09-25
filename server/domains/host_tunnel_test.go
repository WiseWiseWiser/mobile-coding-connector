package domains

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cloudflareSettings "github.com/xhd2015/ai-critic/server/cloudflare"
	"github.com/xhd2015/ai-critic/server/config"
)

// TestStopHostDomainTunnelStopsProxySession covers the seam the owned-domain
// port forward calls on removal. In proxy mode the publish owns an edge mapping,
// so the stop must cancel the session; tearing down a cloudflared route instead
// leaves the hostname published on a dead dial pool.
func TestStopHostDomainTunnelStopsProxySession(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cloudflare.json")
	if err := os.WriteFile(cfgPath, []byte(`{"mode":"proxy"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cloudflareSettings.SetConfigFile(cfgPath)
	t.Cleanup(func() { cloudflareSettings.SetConfigFile(config.CloudflareFile) })

	const domain = "host-stop.example.com"
	cloudflareSettings.SetTestProxySession(domain, true)
	t.Cleanup(func() { cloudflareSettings.SetTestProxySession(domain, false) })
	restoreCounts := cloudflareSettings.SetTestProxyDialCounts(map[string]int{domain: 32}, nil)
	t.Cleanup(restoreCounts)

	if got := cloudflareSettings.GetDomainTunnelStatus(domain); got.Status != "active" {
		t.Fatalf("precondition: status = %q, want %q", got.Status, "active")
	}

	if err := StopHostDomainTunnel(domain, nil); err != nil {
		t.Fatalf("StopHostDomainTunnel() error = %v", err)
	}

	if got := cloudflareSettings.GetDomainTunnelStatus(domain); got.Status == "active" {
		t.Fatalf("publish session survived StopHostDomainTunnel: status = %q", got.Status)
	}
}

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
