package domains

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cloudflareSettings "github.com/xhd2015/ai-critic/server/cloudflare"
	"github.com/xhd2015/ai-critic/server/cloudflare/unified_tunnel"
	"github.com/xhd2015/ai-critic/server/config"
	serverqemu "github.com/xhd2015/ai-critic/server/qemu"
	pkgcf "github.com/xhd2015/dot-pkgs/go-pkgs/cloudflare"
)

func TestBootstrapFakeConnectorProcess(t *testing.T) {
	if os.Getenv("AI_CRITIC_BOOTSTRAP_FAKE_CONNECTOR") != "1" {
		t.Skip("test-owned connector helper")
	}
	time.Sleep(2 * time.Minute) // stopped by the test-owned managed registry
}

// Exercises the actual startup entrypoint, selected host backend, managed
// registry, /service status source, and retry after DNS returns. cloudflared,
// HOME, configuration, connector and probes are all disposable test fixtures.
func TestFreshBootstrapRecoversWithIsolatedCloudflared(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(filepath.Join(home, ".cloudflared"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(root, "network-ready")
	logPath := filepath.Join(root, "commands.log")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("AI_CRITIC_BOOTSTRAP_NETWORK_READY", marker)
	t.Setenv("AI_CRITIC_BOOTSTRAP_LOG", logPath)
	t.Setenv("AI_CRITIC_BOOTSTRAP_TEST_BIN", executable)
	t.Setenv("AI_CRITIC_BOOTSTRAP_FAKE_CONNECTOR", "1")
	const id = "11111111-1111-1111-1111-111111111111"
	if err := os.WriteFile(filepath.Join(home, ".cloudflared", "cert.pem"), []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$AI_CRITIC_BOOTSTRAP_LOG"
case "$*" in
  'tunnel list --output json')
    if [ ! -f "$AI_CRITIC_BOOTSTRAP_NETWORK_READY" ]; then echo 'lookup api.cloudflare.com: no such host' >&2; exit 1; fi
    printf '[{"id":"11111111-1111-1111-1111-111111111111","name":"isolated-bootstrap"}]\n' ;;
  'tunnel info isolated-bootstrap')
    if [ ! -f "$AI_CRITIC_BOOTSTRAP_NETWORK_READY" ]; then echo 'lookup api.cloudflare.com: no such host' >&2; exit 1; fi
    echo 'ID: 11111111-1111-1111-1111-111111111111' ;;
  'tunnel create isolated-bootstrap') echo 'lookup api.cloudflare.com: no such host' >&2; exit 1 ;;
  'tunnel route dns '*) echo 'Added CNAME' ;;
  'tunnel --config '*" run isolated-bootstrap") exec "$AI_CRITIC_BOOTSTRAP_TEST_BIN" -test.run '^TestBootstrapFakeConnectorProcess$' -- "$@" ;;
  *) echo "unexpected test command: $*" >&2; exit 1 ;;
esac
`
	if err := os.WriteFile(filepath.Join(binDir, "cloudflared"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(root, "server-domains.json")
	if err := os.WriteFile(cfgPath, []byte(`{"domains":[{"domain":"bootstrap.example","provider":"cloudflare"}],"tunnel_name":"isolated-bootstrap"}`), 0600); err != nil {
		t.Fatal(err)
	}
	originalPath, originalPort, originalController := getDomainsFile(), getServerPort(), bootstrapDomains
	SetDomainsFile(cfgPath)
	SetServerPort(49235)
	cloudflareSettings.SetConfigFile(filepath.Join(root, "cloudflare.json"))
	serverqemu.SetConfigFile(filepath.Join(root, "qemu.json"))
	r := newDomainReconciler()
	r.retryMin = 100 * time.Millisecond
	r.retryMax = 200 * time.Millisecond
	r.probeInterval = 100 * time.Millisecond
	bootstrapDomains = r
	probe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer probe.Close()
	restoreProbe := unified_tunnel.SetHealthCheckBaseURLForTest(probe.URL)
	t.Cleanup(func() {
		_ = r.stopDomain("bootstrap.example", nil) // only the fixture's managed connector
		_ = pkgcf.Detach(pkgcf.DetachOptions{Domain: "sibling.example", TunnelName: "isolated-bootstrap", ConfigDir: filepath.Join(home, ".cloudflared")})
		r.shutdown()
		restoreProbe()
		bootstrapDomains = originalController
		SetDomainsFile(originalPath)
		SetServerPort(originalPort)
		cloudflareSettings.SetConfigFile(config.CloudflareFile)
		serverqemu.SetConfigFile(config.QemuFile)
	})
	AutoStartTunnels()
	waitDomainCondition(t, func() bool {
		s, ok := r.status("bootstrap.example")
		return ok && s.Status == "error" && strings.Contains(s.Error, "could not determine tunnel ID")
	})
	if err := os.WriteFile(marker, nil, 0600); err != nil {
		t.Fatal(err)
	}
	waitDomainCondition(t, func() bool {
		s, ok := r.status("bootstrap.example")
		return ok && s.Status == "error" && strings.Contains(s.Error, "credentials file not found")
	})
	if _, err := StartHostDomainTunnel("sibling.example", 3080, nil); err == nil || !strings.Contains(err.Error(), "restore credentials") {
		t.Fatalf("sibling missing-credentials error=%v", err)
	}
	cfg, err := LoadDomains()
	if err != nil || cfg.TunnelName != "isolated-bootstrap" {
		t.Fatalf("shared tunnel identity changed: %+v %v", cfg, err)
	}
	if err := os.WriteFile(filepath.Join(home, ".cloudflared", id+".json"), []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	waitDomainCondition(t, func() bool { s, ok := r.status("bootstrap.example"); return ok && s.Status == "active" })
	rows := GetActiveDomainTunnels()
	if len(rows) != 1 || rows[0].Domain != "bootstrap.example" || rows[0].Status != "active" {
		t.Fatalf("service bootstrap status=%+v", rows)
	}
	managed, err := pkgcf.ManagedTunnelDir(filepath.Join(home, ".cloudflared"), "isolated-bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	state, err := pkgcf.LoadTunnelState(managed)
	if err != nil {
		t.Fatal(err)
	}
	if state.Hosts["bootstrap.example"] == nil || state.Hosts["bootstrap.example"].Service != "http://127.0.0.1:49235" || state.ConnectorPID <= 0 {
		t.Fatalf("managed fixture state=%+v", state)
	}
	commands, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(commands), "tunnel route dns --overwrite-dns isolated-bootstrap bootstrap.example") {
		t.Fatalf("test route was not published: %s", commands)
	}
	if strings.Contains(string(commands), "mcc-") {
		t.Fatalf("bootstrap used competing core publisher: %s", commands)
	}
	if _, err := pkgcf.Attach(pkgcf.AttachOptions{Domain: "sibling.example", LocalURL: "http://127.0.0.1:3080", TunnelName: "isolated-bootstrap", ConfigDir: filepath.Join(home, ".cloudflared")}); err != nil {
		t.Fatal(err)
	}
	if err := r.stopDomain("bootstrap.example", nil); err != nil {
		t.Fatal(err)
	}
	state, err = pkgcf.LoadTunnelState(managed)
	if err != nil {
		t.Fatal(err)
	}
	if state.Hosts["bootstrap.example"] != nil || state.Hosts["sibling.example"] == nil || state.Hosts["sibling.example"].Service != "http://127.0.0.1:3080" || state.ConnectorPID <= 0 {
		t.Fatalf("manual stop failed to preserve sibling: %+v", state)
	}
	t.Log(fmt.Sprintf("fresh bootstrap recovered after %s network outage", r.retryMin))
}
