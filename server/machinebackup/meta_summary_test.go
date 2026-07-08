package machinebackup

import (
	"strings"
	"testing"
	"time"
)

func TestFormatInstalledSoftwareSummaryLines(t *testing.T) {
	old := buildInstalledToolsSnapshotFn
	buildInstalledToolsSnapshotFn = func() ([]byte, error) {
		return []byte(`{
  "captured_at": "2026-07-07T12:00:00Z",
  "tools": [
    {"name": "doctest", "path": "/usr/local/bin/doctest"},
    {"name": "agentcli", "version": "1.2.3", "path": "/usr/local/bin/agentcli"}
  ]
}`), nil
	}
	t.Cleanup(func() { buildInstalledToolsSnapshotFn = old })

	text := strings.Join(formatInstalledSoftwareSummaryLines(), "\n")
	for _, want := range []string{
		"INSTALLED SOFTWARE(.backup/installed.json):",
		"captured_at: 2026-07-07T12:00:00Z  (2 tools)",
		"NAME",
		"VERSION",
		"PATH",
		"agentcli             1.2.3",
		"doctest",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("summary missing %q:\n%s", want, text)
		}
	}
}

func TestFormatInstalledSoftwareSummaryLinesNone(t *testing.T) {
	old := buildInstalledToolsSnapshotFn
	buildInstalledToolsSnapshotFn = func() ([]byte, error) {
		return []byte(`{"captured_at":"2026-07-07T12:00:00Z","tools":[]}`), nil
	}
	t.Cleanup(func() { buildInstalledToolsSnapshotFn = old })

	got := strings.Join(formatInstalledSoftwareSummaryLines(), "\n")
	want := "  INSTALLED SOFTWARE(.backup/installed.json): (none)"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFormatEnvSummaryLines(t *testing.T) {
	text := strings.Join(formatEnvSummaryLines(), "\n")
	if !strings.HasPrefix(text, "  ENV(.backup/ENV):\n") {
		t.Fatalf("missing ENV header:\n%s", text)
	}
	if !strings.Contains(text, "    ") {
		t.Fatalf("missing indented env lines:\n%s", text)
	}
	for _, line := range strings.Split(text, "\n")[1:] {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "    ") {
			t.Fatalf("env line not indented: %q", line)
		}
		if !strings.Contains(line, "=") {
			t.Fatalf("env line missing KEY=VALUE: %q", line)
		}
	}
}

func TestFormatMetaCapturedAt(t *testing.T) {
	got := formatMetaCapturedAt(time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC))
	if got != "2026-07-07T12:00:00Z" {
		t.Fatalf("captured_at = %q", got)
	}
}