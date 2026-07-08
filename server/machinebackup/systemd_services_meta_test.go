package machinebackup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFormatSystemdServicesSummaryLines(t *testing.T) {
	capturedAt := time.Date(2026, 7, 7, 10, 15, 0, 0, time.UTC)
	snap := &SystemdServicesSnapshot{
		Version:          systemdServicesVersion,
		CapturedAt:       capturedAt,
		SystemdAvailable: true,
		Scopes: SystemdServicesScopes{
			User: SystemdScopeSnapshot{
				Available:    true,
				RunningCount: 1,
				Units: []SystemdUnitSnapshot{
					{
						Unit:        "agent-proxy.service",
						Load:        "loaded",
						Active:      "active",
						Sub:         "running",
						Description: "AI Critic remote agent proxy",
						MainPID:     4521,
						UnitFile:    "/home/agent/.config/systemd/user/agent-proxy.service",
					},
				},
			},
			System: SystemdScopeSnapshot{
				Available:    true,
				RunningCount: 2,
				Units: []SystemdUnitSnapshot{
					{
						Unit:        "docker.service",
						Load:        "loaded",
						Active:      "active",
						Sub:         "running",
						Description: "Docker Application Container Engine",
						MainPID:     890,
						UnitFile:    "/lib/systemd/system/docker.service",
					},
					{
						Unit:        "tailscaled.service",
						Load:        "loaded",
						Active:      "active",
						Sub:         "running",
						Description: "Tailscale node agent",
						MainPID:     1234,
						UnitFile:    "/lib/systemd/system/tailscaled.service",
					},
				},
			},
		},
	}

	text := strings.Join(formatSystemdServicesSummaryLines(snap), "\n")
	for _, want := range []string{
		"SYSTEMD SERVICES(.backup/systemd-services.json):",
		"captured_at: 2026-07-07T10:15:00Z  (3 running: 1 user, 2 system)",
		"USER (1)",
		"UNIT",
		"PID",
		"DESCRIPTION",
		"agent-proxy.service",
		"4521",
		"AI Critic remote agent proxy",
		"SYSTEM (2)",
		"tailscaled.service",
		"1234",
		"docker.service",
		"890",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("summary missing %q:\n%s", want, text)
		}
	}
}

func TestFormatSystemdServicesSummaryLinesZeroRunning(t *testing.T) {
	snap := &SystemdServicesSnapshot{
		Version:          systemdServicesVersion,
		CapturedAt:       time.Date(2026, 7, 7, 10, 15, 0, 0, time.UTC),
		SystemdAvailable: true,
		Scopes: SystemdServicesScopes{
			User:   SystemdScopeSnapshot{Available: true, Units: []SystemdUnitSnapshot{}},
			System: SystemdScopeSnapshot{Available: true, Units: []SystemdUnitSnapshot{}},
		},
	}

	text := strings.Join(formatSystemdServicesSummaryLines(snap), "\n")
	if !strings.Contains(text, "(0 running)") {
		t.Fatalf("summary missing zero-running marker:\n%s", text)
	}
	for _, forbidden := range []string{"USER (", "SYSTEM ("} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("zero-running summary unexpectedly contains %q:\n%s", forbidden, text)
		}
	}
}

func TestFormatSystemdServicesSummaryLinesScopeError(t *testing.T) {
	snap := &SystemdServicesSnapshot{
		Version:          systemdServicesVersion,
		CapturedAt:       time.Date(2026, 7, 7, 10, 15, 0, 0, time.UTC),
		SystemdAvailable: true,
		Scopes: SystemdServicesScopes{
			User: SystemdScopeSnapshot{
				Available:    true,
				RunningCount: 1,
				Units: []SystemdUnitSnapshot{
					{
						Unit:        "agent-proxy.service",
						Description: "AI Critic remote agent proxy",
						MainPID:     4521,
					},
				},
			},
			System: SystemdScopeSnapshot{
				Available: false,
				Error:     "exit status 1: Access denied",
				Units:     []SystemdUnitSnapshot{},
			},
		},
	}

	text := strings.Join(formatSystemdServicesSummaryLines(snap), "\n")
	for _, want := range []string{
		"(1 running: 1 user; system unavailable)",
		"USER (1)",
		"SYSTEM",
		"(unavailable: Access denied)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("summary missing %q:\n%s", want, text)
		}
	}
}

func TestFormatSystemdServicesSummaryLinesNotAvailableOmits(t *testing.T) {
	if got := formatSystemdServicesSummaryLines(nil); got != nil {
		t.Fatalf("nil snap = %v, want nil", got)
	}
	if got := formatSystemdServicesSummaryLines(&SystemdServicesSnapshot{SystemdAvailable: false}); got != nil {
		t.Fatalf("not available = %v, want nil", got)
	}
}

func TestFormatSystemdServicesSummaryLinesForHomeNotAvailable(t *testing.T) {
	old := buildSystemdServicesSnapshotFn
	buildSystemdServicesSnapshotFn = func(home string) (*SystemdServicesSnapshot, bool, error) {
		return nil, false, nil
	}
	t.Cleanup(func() { buildSystemdServicesSnapshotFn = old })

	if got := formatSystemdServicesSummaryLinesForHome(t.TempDir()); got != nil {
		t.Fatalf("not available home = %v, want nil", got)
	}
}

func TestCaptureSystemdServicesWithHarnessMockScript(t *testing.T) {
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
	start := strings.Index(text, "const systemdMockScript = `")
	if start < 0 {
		t.Fatal("systemdMockScript not found in harness SETUP.md")
	}
	start += len("const systemdMockScript = `")
	end := strings.Index(text[start:], "`\n\nfunc seedSystemdMock")
	if end < 0 {
		t.Fatal("systemdMockScript end not found in harness SETUP.md")
	}
	mock := text[start : start+end]
	if err := os.WriteFile(filepath.Join(binDir, "systemctl"), []byte(mock), 0755); err != nil {
		t.Fatal(err)
	}

	emptyBin := filepath.Join(home, "empty-bin")
	if err := os.MkdirAll(emptyBin, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", emptyBin+string(os.PathListSeparator)+"/usr/bin:/bin")
	if _, err := exec.LookPath("systemctl"); err == nil {
		t.Fatal("expected systemctl absent from process PATH")
	}

	snap, included, err := CaptureSystemdServices(home)
	if err != nil {
		t.Fatalf("CaptureSystemdServices: %v", err)
	}
	if !included || snap == nil {
		t.Fatal("expected included systemd snapshot")
	}
	if !snap.SystemdAvailable {
		t.Fatal("expected systemd_available=true")
	}
	if snap.Scopes.User.RunningCount != 1 {
		t.Fatalf("user running_count = %d, want 1", snap.Scopes.User.RunningCount)
	}
	if snap.Scopes.System.RunningCount != 2 {
		t.Fatalf("system running_count = %d, want 2", snap.Scopes.System.RunningCount)
	}
	if snap.Scopes.User.Units[0].MainPID != 4521 {
		t.Fatalf("user unit main_pid = %d, want 4521", snap.Scopes.User.Units[0].MainPID)
	}
	if snap.Scopes.User.Units[0].UnitFile == "" {
		t.Fatal("user unit missing unit_file from show enrichment")
	}

	lines := formatSystemdServicesSummaryLines(snap)
	summary := strings.Join(lines, "\n")
	if !strings.Contains(summary, "SYSTEMD SERVICES(.backup/systemd-services.json):") {
		t.Fatalf("missing dry-run header:\n%s", summary)
	}
}

func TestCaptureSystemdServicesNotAvailableOmits(t *testing.T) {
	home := t.TempDir()
	emptyBin := filepath.Join(home, "empty-bin")
	if err := os.MkdirAll(emptyBin, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", emptyBin)

	snap, included, err := CaptureSystemdServices(home)
	if err != nil {
		t.Fatalf("CaptureSystemdServices: %v", err)
	}
	if included || snap != nil {
		t.Fatalf("expected omitted snapshot; included=%v snap=%v", included, snap)
	}
}

func TestNormalizeHarnessSystemctlScriptFixesIndentedHeredoc(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	indented := "#!/bin/sh\n\tset -e\n\tif [ \"$1\" = \"--version\" ]; then echo systemd; exit 0; fi\n\tcase \"$*\" in\n\t*list-units*--type=service*--state=running*--output=json*)\n\t  cat <<'MOCK_EOF'\n\t[]\n\tMOCK_EOF\n\t  exit 0\n\t  ;;\n\tesac\n\texit 1\n"
	bin := filepath.Join(binDir, "systemctl")
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
	out, err := exec.Command("sh", execPath, "--user", "list-units", "--type=service", "--state=running", "--output=json").CombinedOutput()
	if err != nil {
		t.Fatalf("indented harness mock failed: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) != "[]" {
		t.Fatalf("unexpected list-units output: %s", out)
	}
}

func TestCaptureSystemdServicesWithEmptyMock(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	mock := `#!/bin/sh
set -e
if [ "$1" = "--version" ]; then
  echo "systemd 252"
  exit 0
fi
if [ "${SYSTEMD_MOCK_EMPTY:-0}" = "1" ]; then
  printf '%s\n' '[]'
  exit 0
fi
exit 1
`
	if err := os.WriteFile(filepath.Join(binDir, "systemctl"), []byte(mock), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SYSTEMD_MOCK_EMPTY", "1")

	emptyBin := filepath.Join(home, "empty-bin")
	if err := os.MkdirAll(emptyBin, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", emptyBin+string(os.PathListSeparator)+"/usr/bin:/bin")

	snap, included, err := CaptureSystemdServices(home)
	if err != nil {
		t.Fatalf("CaptureSystemdServices: %v", err)
	}
	if !included || snap == nil {
		t.Fatal("expected included snapshot for empty mock")
	}
	if snap.Scopes.User.RunningCount != 0 || snap.Scopes.System.RunningCount != 0 {
		t.Fatalf("expected zero running counts; user=%d system=%d", snap.Scopes.User.RunningCount, snap.Scopes.System.RunningCount)
	}
}