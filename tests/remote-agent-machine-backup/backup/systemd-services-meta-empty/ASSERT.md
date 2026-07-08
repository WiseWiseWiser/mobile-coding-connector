## Expected Output

Dry-run summary includes `SYSTEMD SERVICES(.backup/systemd-services.json):` with
`captured_at` subheader `(0 running)` and no USER/SYSTEM unit tables. Archive contains
`systemd-services.json` with `running_count: 0` for both scopes. Stdout ends with `\n`.

## Expected

1. Exit code 0.
2. Dry-run combined output has SYSTEMD SERVICES section with `(0 running)`.
3. SYSTEMD section has no USER or SYSTEM unit table rows.
4. Archive lists `.backup/systemd-services.json` with zero running counts.
5. Stdout ends with `\n`.

## Side Effects

Creates `systemd-services-meta-empty.tar.xz` under `agentHome`.

## Errors

- Missing archive member when gate passes.
- Non-zero running counts in JSON or summary.

## Exit Code

0.

```go
import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

var systemdZeroCapturedAtRE = regexp.MustCompile(`captured_at: .+  \(0 running\)`)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if resp.BackupPath == "" {
		t.Fatal("BackupPath empty")
	}
	if _, err := os.Stat(resp.BackupPath); err != nil {
		t.Fatalf("backup file missing: %v", err)
	}

	combined := resp.DryRunCombined
	if combined == "" {
		combined = resp.Combined
	}
	assertMetaSectionsBeforeTotal(t, combined)

	systemdSection := systemdServicesSummarySection(combined)
	if systemdSection == "" {
		t.Fatalf("missing SYSTEMD SERVICES summary section; got:\n%s", combined)
	}
	header := metaSectionHeaderLines(systemdSection, 1)
	assert.Output(t, header, `---
version: 2
---
SYSTEMD SERVICES(.backup/systemd-services.json):
`)

	if !systemdZeroCapturedAtRE.MatchString(systemdSection) {
		t.Fatalf("SYSTEMD section missing zero-running subheader; section:\n%s", systemdSection)
	}

	if strings.Contains(systemdSection, "USER (") || strings.Contains(systemdSection, "SYSTEM (") {
		t.Fatalf("SYSTEMD section unexpectedly has scope tables for zero running; section:\n%s", systemdSection)
	}
	for _, unit := range []string{systemdFixtureUserUnit, systemdFixtureTailscaledUnit, systemdFixtureDockerUnit} {
		if strings.Contains(systemdSection, unit) {
			t.Fatalf("SYSTEMD section unexpectedly lists unit %q for zero running; section:\n%s", unit, systemdSection)
		}
	}

	archiveHasXZMagic(t, resp.BackupPath)
	members := tarXZListMembers(t, resp.BackupPath)
	if !memberListContains(members, ".backup/systemd-services.json") {
		t.Fatalf("archive missing .backup/systemd-services.json; members=%v", members)
	}

	raw := tarXZExtractFile(t, resp.BackupPath, ".backup/systemd-services.json")
	snap := parseSystemdServicesJSON(t, raw)
	assertSystemdServicesBasics(t, snap)
	assertSystemdServicesRunningCounts(t, snap, 0, 0)
	if len(snap.Scopes.User.Units) != 0 || len(snap.Scopes.System.Units) != 0 {
		t.Fatalf("zero-running snapshot has units: user=%v system=%v", snap.Scopes.User.Units, snap.Scopes.System.Units)
	}

	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```