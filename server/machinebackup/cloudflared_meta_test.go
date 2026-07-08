package machinebackup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRedactCloudflaredConfigYAML(t *testing.T) {
	raw := `tunnel: fake-tunnel-id-should-redact
credentials-file: fake-tunnel-id.json
ingress:
  - hostname: example.test
    service: http://localhost:8080
`
	redacted := redactCloudflaredConfigYAML(raw)
	for _, forbidden := range []string{"fake-tunnel-id-should-redact", "fake-tunnel-id.json"} {
		if strings.Contains(redacted, forbidden) {
			t.Fatalf("redacted config still contains %q:\n%s", forbidden, redacted)
		}
	}
	if !strings.Contains(strings.ToLower(redacted), "redact") {
		t.Fatalf("redacted config missing redaction marker:\n%s", redacted)
	}
	if !strings.Contains(redacted, "ingress:") {
		t.Fatalf("expected non-sensitive field preserved:\n%s", redacted)
	}
}

func TestFormatCloudflaredSummaryLines(t *testing.T) {
	snap := &CloudflaredConfigSnapshot{
		Version:    cloudflaredConfigVersion,
		CapturedAt: mustParseMetaTime(t, "2026-07-08T10:00:00Z"),
		Running:    true,
		VersionInfo: CloudflaredVersionInfo{
			Text: "cloudflared 2026.1.2",
		},
		Process: CloudflaredProcessInfo{
			Cmdline: "cloudflared tunnel --url http://127.0.0.1:23712",
		},
		QuickTunnel: CloudflaredQuickTunnelInfo{
			URL: "http://127.0.0.1:23712",
		},
		Config: CloudflaredConfigFileInfo{
			Path:    "/home/agent/.cloudflared/config.yml",
			Present: true,
		},
		Setup: CloudflaredSetupInfo{
			BashHistory: []string{"cloudflared tunnel --url http://127.0.0.1:23712"},
		},
	}

	text := strings.Join(formatCloudflaredSummaryLines(snap), "\n")
	for _, want := range []string{
		"CLOUDFLARED(.backup/cloudflared-config.json):",
		"captured_at: 2026-07-08T10:00:00Z  (running)",
		"VERSION",
		"MODE",
		"TARGET",
		"cloudflared 2026.1.2",
		"quick-tunnel",
		"http://127.0.0.1:23712",
		"DAEMON",
		"cloudflared tunnel --url http://127.0.0.1:23712",
		"CONFIG",
		"/home/agent/.cloudflared/config.yml  (present, redacted)",
		"SHELL HISTORY (cloudflared)",
		"[bash] cloudflared tunnel --url http://127.0.0.1:23712",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("summary missing %q:\n%s", want, text)
		}
	}
}

func TestFormatCloudflaredSummaryLinesNotRunningOmits(t *testing.T) {
	if got := formatCloudflaredSummaryLines(nil); got != nil {
		t.Fatalf("nil snap = %v, want nil", got)
	}
	if got := formatCloudflaredSummaryLines(&CloudflaredConfigSnapshot{Running: false}); got != nil {
		t.Fatalf("not running = %v, want nil", got)
	}
}

func TestFormatCloudflaredSummaryLinesForHomeNotRunning(t *testing.T) {
	old := buildCloudflaredConfigSnapshotFn
	buildCloudflaredConfigSnapshotFn = func(home string) (*CloudflaredConfigSnapshot, bool, error) {
		return nil, false, nil
	}
	t.Cleanup(func() { buildCloudflaredConfigSnapshotFn = old })

	if got := formatCloudflaredSummaryLinesForHome(t.TempDir()); got != nil {
		t.Fatalf("not running home = %v, want nil", got)
	}
}

func TestParseCloudflaredQuickTunnel(t *testing.T) {
	info := parseCloudflaredQuickTunnel("cloudflared tunnel --url http://127.0.0.1:23712 --hostname example.test")
	if info.URL != "http://127.0.0.1:23712" {
		t.Fatalf("url = %q", info.URL)
	}
	if info.Hostname != "example.test" {
		t.Fatalf("hostname = %q", info.Hostname)
	}
}

func TestReadCloudflaredHistoryLines(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, home, ".bash_history", "ls\ncloudflared tunnel --url http://127.0.0.1:23712\n")
	writeTestFile(t, home, ".zsh_history", "cd ~\nCLOUDFLARED version\n")

	bash := readCloudflaredHistoryLines(home, ".bash_history")
	if len(bash) != 1 || !strings.Contains(strings.ToLower(bash[0]), "cloudflared") {
		t.Fatalf("bash history = %v", bash)
	}
	zsh := readCloudflaredHistoryLines(home, ".zsh_history")
	if len(zsh) != 1 || !strings.Contains(strings.ToLower(zsh[0]), "cloudflared") {
		t.Fatalf("zsh history = %v", zsh)
	}
}

func TestCaptureCloudflaredConfigWithHarnessMockScript(t *testing.T) {
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
	start := strings.Index(text, "const cloudflaredMockScript = `")
	if start < 0 {
		t.Fatal("cloudflaredMockScript not found in harness SETUP.md")
	}
	start += len("const cloudflaredMockScript = `")
	end := strings.Index(text[start:], "`\n\nconst cloudflaredMockPgrepScript")
	if end < 0 {
		t.Fatal("cloudflaredMockScript end not found in harness SETUP.md")
	}
	mock := text[start : start+end]
	if err := os.WriteFile(filepath.Join(binDir, "cloudflared"), []byte(mock), 0755); err != nil {
		t.Fatal(err)
	}

	pgrepStart := strings.Index(text, "const cloudflaredMockPgrepScript = `")
	if pgrepStart < 0 {
		t.Fatal("cloudflaredMockPgrepScript not found in harness SETUP.md")
	}
	pgrepStart += len("const cloudflaredMockPgrepScript = `")
	pgrepEnd := strings.Index(text[pgrepStart:], "`\n\nconst cloudflaredFixtureConfigYAML")
	if pgrepEnd < 0 {
		t.Fatal("cloudflaredMockPgrepScript end not found in harness SETUP.md")
	}
	pgrepMock := text[pgrepStart : pgrepStart+pgrepEnd]
	if err := os.WriteFile(filepath.Join(binDir, "pgrep"), []byte(pgrepMock), 0755); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, home, ".doctest-cloudflared.pid", "4567\n")
	writeTestFile(t, home, ".doctest-cloudflared.cmdline", "cloudflared tunnel --url http://127.0.0.1:23712")
	writeTestFile(t, home, ".cloudflared/config.yml", "tunnel: secret\ncredentials-file: secret.json\n")
	writeTestFile(t, home, ".bash_history", "cloudflared tunnel --url http://127.0.0.1:23712\n")

	emptyBin := filepath.Join(home, "empty-bin")
	if err := os.MkdirAll(emptyBin, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", emptyBin+string(os.PathListSeparator)+"/usr/bin:/bin")

	snap, included, err := CaptureCloudflaredConfig(home)
	if err != nil {
		t.Fatalf("CaptureCloudflaredConfig: %v", err)
	}
	if !included || snap == nil {
		t.Fatalf("expected included snapshot; home=%s", home)
	}
	if snap.Tunnels.Available {
		t.Fatal("expected tunnels.available=false without credentials")
	}
	if !strings.Contains(snap.Config.RedactedYAML, "redact") {
		t.Fatalf("config not redacted: %s", snap.Config.RedactedYAML)
	}
}

func TestCloudflaredVersionOutputIgnoresNonJSONVersionFlag(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	mock := `#!/bin/sh
case "$1" in
version)
  echo "cloudflared 2026.1.2"
  ;;
esac
`
	if err := os.WriteFile(filepath.Join(binDir, "cloudflared"), []byte(mock), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "pgrep"), []byte("#!/bin/sh\necho 1\n"), 0755); err != nil {
		t.Fatal(err)
	}

	text, raw, err := cloudflaredVersionOutput(home)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(text) != "cloudflared 2026.1.2" {
		t.Fatalf("text = %q", text)
	}
	if raw != nil {
		t.Fatalf("json = %q, want nil for non-json version output", string(raw))
	}
}

func TestCaptureCloudflaredConfigWithMockCLI(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	mock := `#!/bin/sh
set -e
case "$1" in
version)
  echo "cloudflared 2026.1.2"
  ;;
tunnel)
  if [ "$2" = "list" ] && [ "$4" = "json" ]; then
    printf '%s\n' '[]'
    exit 0
  fi
  exit 1
  ;;
*)
  exit 1
  ;;
esac
`
	if err := os.WriteFile(filepath.Join(binDir, "cloudflared"), []byte(mock), 0755); err != nil {
		t.Fatal(err)
	}
	pgrepMock := `#!/bin/sh
echo 4567
`
	if err := os.WriteFile(filepath.Join(binDir, "pgrep"), []byte(pgrepMock), 0755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, home, ".doctest-cloudflared.cmdline", "cloudflared tunnel --url http://127.0.0.1:23712")
	writeTestFile(t, home, ".bash_history", "cloudflared tunnel --url http://127.0.0.1:23712\n")

	emptyBin := filepath.Join(home, "empty-bin")
	if err := os.MkdirAll(emptyBin, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", emptyBin+string(os.PathListSeparator)+"/usr/bin:/bin")
	if _, err := exec.LookPath("cloudflared"); err == nil {
		t.Fatal("expected cloudflared absent from process PATH")
	}

	snap, included, err := CaptureCloudflaredConfig(home)
	if err != nil {
		t.Fatalf("CaptureCloudflaredConfig: %v", err)
	}
	if !included || snap == nil {
		t.Fatal("expected included cloudflared snapshot")
	}
	lines := formatCloudflaredSummaryLines(snap)
	text := strings.Join(lines, "\n")
	if !strings.Contains(text, "CLOUDFLARED(.backup/cloudflared-config.json):") {
		t.Fatalf("missing dry-run header:\n%s", text)
	}
}

func TestCloudflaredRunningGateRequiresHomeSpecificEvidence(t *testing.T) {
	home := t.TempDir()
	emptyBin := filepath.Join(home, "empty-bin")
	if err := os.MkdirAll(emptyBin, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", emptyBin+string(os.PathListSeparator)+"/usr/bin:/bin")

	if cloudflaredRunningGate(home) {
		t.Fatal("expected gate false with no home-specific cloudflared evidence")
	}
}

func TestCloudflaredRunningGateWithHomeConfigFootprint(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	mock := `#!/bin/sh
case "$1" in
version)
  echo "cloudflared 2026.1.2"
  ;;
esac
`
	if err := os.WriteFile(filepath.Join(binDir, "cloudflared"), []byte(mock), 0755); err != nil {
		t.Fatal(err)
	}
	pgrepMock := `#!/bin/sh
echo 4567
`
	if err := os.WriteFile(filepath.Join(binDir, "pgrep"), []byte(pgrepMock), 0755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, home, ".cloudflared/config.yml", "tunnel: secret\ncredentials-file: secret.json\n")
	writeTestFile(t, home, ".doctest-cloudflared.cmdline", "cloudflared tunnel --url http://127.0.0.1:23712")

	emptyBin := filepath.Join(home, "empty-bin")
	if err := os.MkdirAll(emptyBin, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", emptyBin+string(os.PathListSeparator)+"/usr/bin:/bin")

	if !cloudflaredRunningGate(home) {
		t.Fatal("expected gate true with home config footprint and mock process")
	}
}

func TestCloudflaredHomeSpecificEvidence(t *testing.T) {
	home := t.TempDir()
	if cloudflaredHomeSpecificEvidence(home, "") {
		t.Fatal("expected false without home evidence")
	}

	writeTestFile(t, home, ".cloudflared/cert.pem", "fake-cert\n")
	if !cloudflaredHomeSpecificEvidence(home, "") {
		t.Fatal("expected true with ~/.cloudflared/cert.pem")
	}

	otherHome := t.TempDir()
	cmdline := filepath.Join(otherHome, ".cloudflared", "config.yml")
	if !cloudflaredHomeSpecificEvidence(otherHome, "cloudflared --config "+cmdline) {
		t.Fatal("expected true when cmdline contains home path")
	}
}

func mustParseMetaTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}