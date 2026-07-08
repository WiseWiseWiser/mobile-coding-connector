package machinebackup

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRedactTailscalePrefs(t *testing.T) {
	raw := []byte(`{
  "PrivateNodeKey": "nodekey:secret",
  "OldPrivateNodeKey": "nodekey:old",
  "NetworkLockKey": "nlkey:lock",
  "Config": {
    "PrivateNodeKey": "nodekey:nested"
  },
  "AdvertiseRoutes": null
}`)
	redacted, err := redactTailscalePrefs(raw)
	if err != nil {
		t.Fatal(err)
	}
	text := string(redacted)
	for _, forbidden := range []string{
		"nodekey:secret",
		"nodekey:old",
		"nlkey:lock",
		"nodekey:nested",
		"PrivateNodeKey",
		"OldPrivateNodeKey",
		"NetworkLockKey",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("redacted prefs still contains %q:\n%s", forbidden, text)
		}
	}
	if !strings.Contains(text, "AdvertiseRoutes") {
		t.Fatalf("expected non-sensitive field preserved:\n%s", text)
	}
}

func TestFormatTailscaleSummaryLines(t *testing.T) {
	capturedAt := time.Date(2026, 7, 7, 8, 51, 0, 0, time.UTC)
	snap := &TailscaleConfigSnapshot{
		Version:    tailscaleConfigVersion,
		CapturedAt: capturedAt,
		Running:    true,
		VersionInfo: TailscaleVersionInfo{
			Text: "1.96.2",
		},
		Daemon: TailscaleDaemonInfo{
			Cmdline:             "tailscaled --tun=userspace-networking --socks5-server=localhost:1055",
			UserspaceNetworking: true,
			Socks5Server:        "localhost:1055",
		},
		Status: json.RawMessage(`{
  "BackendState": "Running",
  "Self": {
    "TailscaleIPs": ["100.64.209.66"],
    "DNSName": "samd-agent.example.ts.net"
  },
  "Peer": {
    "peerA": {
      "DNSName": "peer-a",
      "TailscaleIPs": ["100.69.30.59"],
      "OS": "linux",
      "Online": false,
      "LastSeen": "2026-07-06T08:51:00Z"
    },
    "peerB": {
      "DNSName": "peer-b",
      "TailscaleIPs": ["100.126.43.79"],
      "OS": "macOS",
      "Online": true
    }
  }
}`),
		Setup: TailscaleSetupInfo{
			Steps: []string{
				"1. Install (proxy if needed): curl -fsSL https://tailscale.com/install.sh | sh",
				"2. Start daemon: tailscaled --tun=userspace-networking --socks5-server=localhost:1055 &",
				"3. Join: tailscale up",
				"4. Verify: tailscale status",
			},
			BashHistory: []string{"tailscale up"},
			ZshHistory:  []string{"tailscaled --tun=userspace-networking --socks5-server=localhost:1055"},
		},
	}

	text := strings.Join(formatTailscaleSummaryLines(snap), "\n")
	for _, want := range []string{
		"TAILSCALE(.backup/tailscale-config.json):",
		"captured_at: 2026-07-07T08:51:00Z  (running)",
		"VERSION",
		"MODE",
		"SOCKS5",
		"TAILSCALE IP",
		"MAGIC DNS",
		"1.96.2",
		"userspace-networking",
		"localhost:1055",
		"100.64.209.66",
		"samd-agent.example.ts.net",
		"DAEMON",
		"tailscaled --tun=userspace-networking --socks5-server=localhost:1055",
		"SETUP",
		"SHELL HISTORY (tailscale)",
		"[bash] tailscale up",
		"[zsh]  tailscaled --tun=userspace-networking --socks5-server=localhost:1055",
		"PEERS (2)",
		"peer-a",
		"100.69.30.59",
		"linux",
		"offline, last seen 1d ago",
		"peer-b",
		"100.126.43.79",
		"macOS",
		"active",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("summary missing %q:\n%s", want, text)
		}
	}
}

func TestFormatTailscaleSummaryLinesNotRunningOmits(t *testing.T) {
	if got := formatTailscaleSummaryLines(nil); got != nil {
		t.Fatalf("nil snap = %v, want nil", got)
	}
	if got := formatTailscaleSummaryLines(&TailscaleConfigSnapshot{Running: false}); got != nil {
		t.Fatalf("not running = %v, want nil", got)
	}
}

func TestFormatTailscaleSummaryLinesForHomeNotRunning(t *testing.T) {
	old := buildTailscaleConfigSnapshotFn
	buildTailscaleConfigSnapshotFn = func(home string) (*TailscaleConfigSnapshot, bool, error) {
		return nil, false, nil
	}
	t.Cleanup(func() { buildTailscaleConfigSnapshotFn = old })

	if got := formatTailscaleSummaryLinesForHome(t.TempDir()); got != nil {
		t.Fatalf("not running home = %v, want nil", got)
	}
}

func TestCaptureTailscaleConfigWithHarnessMockScript(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	setupMD, err := os.ReadFile(filepath.Join("..", "..", "tests", "remote-agent-machine-backup", "SETUP.md"))
	if err != nil {
		t.Skipf("harness SETUP.md not available from package dir: %v", err)
	}
	text := string(setupMD)
	start := strings.Index(text, "const tailscaleMockScript = `")
	if start < 0 {
		t.Fatal("tailscaleMockScript not found in harness SETUP.md")
	}
	start += len("const tailscaleMockScript = `")
	end := strings.Index(text[start:], "`\n\nfunc prependPathToEnv")
	if end < 0 {
		t.Fatal("tailscaleMockScript end not found in harness SETUP.md")
	}
	mock := text[start : start+end]
	if err := os.WriteFile(filepath.Join(binDir, "tailscale"), []byte(mock), 0755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, home, ".bash_history", "ls\ntailscale up\n")
	writeTestFile(t, home, ".zsh_history", "cd ~\ntailscaled --tun=userspace-networking --socks5-server=localhost:1055\n")

	emptyBin := filepath.Join(home, "empty-bin")
	if err := os.MkdirAll(emptyBin, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", emptyBin+string(os.PathListSeparator)+"/usr/bin:/bin")

	snap, included, err := CaptureTailscaleConfig(home)
	if err != nil {
		t.Fatalf("CaptureTailscaleConfig: %v", err)
	}
	if !included || snap == nil {
		bin, ok := tailscaleBinaryInHomeBin(home)
		t.Fatalf("expected included snapshot; home=%s bin=%q ok=%v", home, bin, ok)
	}
}

func TestCaptureTailscaleConfigWithMockCLI(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	mock := `#!/bin/sh
set -e
case "$1" in
version)
  if [ "$2" = "--json" ]; then
    printf '%s\n' '{"ClientVersion":"1.96.2","TUN":true}'
  else
    echo "1.96.2"
  fi
  ;;
status)
  if [ "$2" = "--json" ]; then
    cat <<'MOCK_EOF'
{"BackendState":"Running","Self":{"TailscaleIPs":["100.64.209.66"],"DNSName":"samd-agent.example.ts.net"},"Peer":{"peerA":{"DNSName":"peer-a","TailscaleIPs":["100.69.30.59"],"OS":"linux","Online":false,"LastSeen":"2026-07-06T08:51:00Z"},"peerB":{"DNSName":"peer-b","TailscaleIPs":["100.126.43.79"],"OS":"macOS","Online":true}}}
MOCK_EOF
  else
    exit 1
  fi
  ;;
debug)
  if [ "$2" = "prefs" ]; then
    echo '{"PrivateNodeKey":"nodekey:fake-private-should-redact","Config":{"PrivateNodeKey":"nodekey:fake-nested-should-redact"}}'
  else
    exit 1
  fi
  ;;
*)
  exit 1
  ;;
esac
`
	if err := os.WriteFile(filepath.Join(binDir, "tailscale"), []byte(mock), 0755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, home, ".bash_history", "tailscale up\n")
	writeTestFile(t, home, ".zsh_history", "tailscaled --tun=userspace-networking --socks5-server=localhost:1055\n")

	// PATH lacks tailscale but keeps standard utilities for the shell mock script.
	emptyBin := filepath.Join(home, "empty-bin")
	if err := os.MkdirAll(emptyBin, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", emptyBin+string(os.PathListSeparator)+"/usr/bin:/bin")
	if _, err := exec.LookPath("tailscale"); err == nil {
		t.Fatal("expected tailscale absent from process PATH")
	}

	snap, included, err := CaptureTailscaleConfig(home)
	if err != nil {
		t.Fatalf("CaptureTailscaleConfig: %v", err)
	}
	if !included || snap == nil {
		t.Fatal("expected included tailscale snapshot")
	}
	lines := formatTailscaleSummaryLines(snap)
	text := strings.Join(lines, "\n")
	if !strings.Contains(text, "TAILSCALE(.backup/tailscale-config.json):") {
		t.Fatalf("missing dry-run header:\n%s", text)
	}
}

func TestReadTailscaleHistoryLines(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, home, ".bash_history", "ls\ntailscale up\n")
	writeTestFile(t, home, ".zsh_history", "cd ~\nTAILSCALED --socks5-server=localhost:1055\n")

	bash := readTailscaleHistoryLines(home, ".bash_history")
	if len(bash) != 1 || bash[0] != "tailscale up" {
		t.Fatalf("bash history = %v", bash)
	}
	zsh := readTailscaleHistoryLines(home, ".zsh_history")
	if len(zsh) != 1 || !strings.Contains(strings.ToLower(zsh[0]), "tailscaled") {
		t.Fatalf("zsh history = %v", zsh)
	}
}

func TestApplyDaemonFlagParsing(t *testing.T) {
	info := TailscaleDaemonInfo{Cmdline: "tailscaled --tun=userspace-networking --socks5-server=localhost:1055 --state=/var/lib/tailscale/tailscaled.state --socket=/var/run/tailscale/tailscaled.sock"}
	applyDaemonFlagParsing(&info)
	if !info.UserspaceNetworking {
		t.Fatal("expected userspace networking")
	}
	if info.Socks5Server != "localhost:1055" {
		t.Fatalf("socks5 = %q", info.Socks5Server)
	}
	if info.StatePath != "/var/lib/tailscale/tailscaled.state" {
		t.Fatalf("state = %q", info.StatePath)
	}
	if info.SocketPath != "/var/run/tailscale/tailscaled.sock" {
		t.Fatalf("socket = %q", info.SocketPath)
	}
}

func TestDiscoverTailscaleDaemonFallsBackToHistoryWhenProcCmdlineEmpty(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, home, ".zsh_history", "cd ~\ntailscaled --tun=userspace-networking --socks5-server=localhost:1055\n")

	info := discoverTailscaleDaemon(home)
	if info.Cmdline == "" {
		t.Fatal("expected cmdline from shell history fallback")
	}
	if !strings.Contains(info.Cmdline, "userspace-networking") {
		t.Fatalf("cmdline = %q", info.Cmdline)
	}
	if !info.UserspaceNetworking {
		t.Fatal("expected userspace-networking from history cmdline")
	}
	if info.Socks5Server != "localhost:1055" {
		t.Fatalf("socks5 = %q", info.Socks5Server)
	}
}

func TestNormalizeHarnessTailscaleScriptFixesIndentedHeredoc(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	indented := "#!/bin/sh\n\tset -e\n\tcase \"$1\" in\n\tstatus)\n\t  if [ \"$2\" = \"--json\" ]; then\n\t    cat <<'MOCK_EOF'\n\t{\"BackendState\":\"Running\"}\n\tMOCK_EOF\n\t  else exit 1; fi\n\t  ;;\n\tesac\n"
	bin := filepath.Join(binDir, "tailscale")
	if err := os.WriteFile(bin, []byte(indented), 0755); err != nil {
		t.Fatal(err)
	}
	execPath, cleanup, err := normalizeHarnessTailscaleScript(bin)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if execPath == bin {
		t.Fatal("expected rewritten temp script")
	}
	out, err := exec.Command("sh", execPath, "status", "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("indented harness mock failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), `"BackendState":"Running"`) {
		t.Fatalf("unexpected status output: %s", out)
	}
}

func TestCaptureTailscaleConfigWithNonExecutableMockCLI(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	mock := `#!/bin/sh
set -e
case "$1" in
status)
  if [ "$2" = "--json" ]; then
    echo '{"BackendState":"Running","Self":{"TailscaleIPs":["100.64.209.66"],"DNSName":"samd-agent.example.ts.net"},"Peer":{}}'
  else
    exit 1
  fi
  ;;
version)
  if [ "$2" = "--json" ]; then
    echo '{"ClientVersion":"1.96.2","TUN":true}'
  else
    echo "1.96.2"
  fi
  ;;
debug)
  if [ "$2" = "prefs" ]; then
    echo '{"Config":{}}'
  else
    exit 1
  fi
  ;;
*)
  exit 1
  ;;
esac
`
	tailscaleBin := filepath.Join(binDir, "tailscale")
	if err := os.WriteFile(tailscaleBin, []byte(mock), 0644); err != nil {
		t.Fatal(err)
	}

	emptyBin := filepath.Join(home, "empty-bin")
	if err := os.MkdirAll(emptyBin, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", emptyBin+string(os.PathListSeparator)+"/usr/bin:/bin")

	snap, included, err := CaptureTailscaleConfig(home)
	if err != nil {
		t.Fatalf("CaptureTailscaleConfig: %v", err)
	}
	if !included || snap == nil {
		t.Fatal("expected included tailscale snapshot for non-executable mock in HOME/bin")
	}
}

