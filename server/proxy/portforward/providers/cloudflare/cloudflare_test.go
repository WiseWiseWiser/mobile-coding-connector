package cloudflare

import (
	"os"
	"path/filepath"
	"testing"

	cfutils "github.com/xhd2015/ai-critic/server/cloudflare"
	"github.com/xhd2015/ai-critic/server/config"
)

// useProxyMode points the cloudflare config at a temp file in proxy mode, so
// teardown takes the proxy branch without touching the real config.
func useProxyMode(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "cloudflare.json")
	if err := os.WriteFile(path, []byte(`{"mode":"proxy","proxy_url":"http://127.0.0.1:1","token":"t"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfutils.SetConfigFile(path)
	t.Cleanup(func() { cfutils.SetConfigFile(config.CloudflareFile) })
	if !cfutils.ProxyModeEnabled() {
		t.Fatal("precondition: proxy mode should be enabled")
	}
}

// TestStopOwnedForwardStopsProxySession pins the defect that stranded a
// published hostname after every service restart: removing an owned-domain
// forward tore down the cloudflared backend route but left the proxy-mode
// publish session running, so the edge kept a mapping whose dial pool was dead
// while status still reported "active".
//
// The session must be gone once the forward is stopped; otherwise the next
// Start sees "active" (a live dial pool on the edge) and reuses the wedged
// session instead of publishing a fresh mapping.
func TestStopOwnedForwardStopsProxySession(t *testing.T) {
	const domain = "owned-stop.example.com"
	useProxyMode(t)

	cfutils.SetTestProxySession(domain, true)
	t.Cleanup(func() { cfutils.SetTestProxySession(domain, false) })
	// A live dial pool is what made the stale session look healthy.
	restoreCounts := cfutils.SetTestProxyDialCounts(map[string]int{domain: 32}, nil)
	t.Cleanup(restoreCounts)

	if got := cfutils.GetDomainTunnelStatus(domain); got.Status != "active" {
		t.Fatalf("precondition: status = %q, want %q", got.Status, "active")
	}

	stopOwnedForward(domain, nil)

	if got := cfutils.GetDomainTunnelStatus(domain); got.Status == "active" {
		t.Fatalf("publish session survived the forward stop: status = %q", got.Status)
	}
}
