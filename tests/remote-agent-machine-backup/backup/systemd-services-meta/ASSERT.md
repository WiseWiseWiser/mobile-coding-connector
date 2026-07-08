## Expected Output

Dry-run summary includes `SYSTEMD SERVICES(.backup/systemd-services.json):` after ENV and
before TOTAL with `captured_at` subheader `(3 running: 1 user, 2 system)`, USER table
listing `agent-proxy.service`, and SYSTEM table listing `tailscaled.service` and
`docker.service`. Archive contains valid `systemd-services.json` with version `1.0` and
per-scope running counts. Stdout ends with `\n`.

## Expected

1. Exit code 0.
2. Dry-run combined output has SYSTEMD SERVICES section after ENV, before TOTAL.
3. SYSTEMD section contains USER and SYSTEM subtables with UNIT/PID/DESCRIPTION headers.
4. SYSTEMD section lists mock user unit and both system units with fixture PIDs/descriptions.
5. Archive lists `.backup/systemd-services.json`.
6. Archive JSON version `1.0`, `systemd_available: true`, running counts 1 user / 2 system.
7. Stdout ends with `\n`.

## Side Effects

Creates `systemd-services-meta.tar.xz` under `agentHome`.

## Errors

- Missing SYSTEMD section or archive member when mock is seeded.
- Wrong section order relative to TOTAL.

## Exit Code

0.

```go
import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

var (
	systemdCapturedAtRE  = regexp.MustCompile(`captured_at: .+  \(3 running: 1 user, 2 system\)`)
	systemdUserScopeRE   = regexp.MustCompile(`USER \(\d+\)`)
	systemdSystemScopeRE = regexp.MustCompile(`SYSTEM \(\d+\)`)
	systemdUserUnitRowRE = regexp.MustCompile(regexp.QuoteMeta(systemdFixtureUserUnit) + `.*` + strconv.Itoa(systemdFixtureUserPID) + `.*` + regexp.QuoteMeta(systemdFixtureUserDesc))
	systemdDockerRowRE   = regexp.MustCompile(regexp.QuoteMeta(systemdFixtureDockerUnit) + `.*` + strconv.Itoa(systemdFixtureDockerPID) + `.*` + regexp.QuoteMeta(systemdFixtureDockerDesc))
	systemdTailscaledRowRE = regexp.MustCompile(regexp.QuoteMeta(systemdFixtureTailscaledUnit) + `.*` + strconv.Itoa(systemdFixtureTailscaledPID) + `.*` + regexp.QuoteMeta(systemdFixtureTailscaledDesc))
)

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

	if !systemdCapturedAtRE.MatchString(systemdSection) {
		t.Fatalf("SYSTEMD section missing running-count subheader; section:\n%s", systemdSection)
	}

	if !systemdUserScopeRE.MatchString(systemdSection) || !systemdSystemScopeRE.MatchString(systemdSection) {
		t.Fatalf("SYSTEMD section missing USER/SYSTEM scope headers; section:\n%s", systemdSection)
	}
	if !systemdUserUnitRowRE.MatchString(systemdSection) {
		t.Fatalf("SYSTEMD section missing user unit row; section:\n%s", systemdSection)
	}
	if !systemdDockerRowRE.MatchString(systemdSection) || !systemdTailscaledRowRE.MatchString(systemdSection) {
		t.Fatalf("SYSTEMD section missing system unit rows; section:\n%s", systemdSection)
	}

	assertSystemdServicesTableHeaders(t, systemdSection)
	for _, needle := range []string{"USER", "SYSTEM", systemdFixtureUserUnit, systemdFixtureTailscaledUnit, systemdFixtureDockerUnit} {
		if !strings.Contains(systemdSection, needle) {
			t.Fatalf("SYSTEMD section missing %q; section:\n%s", needle, systemdSection)
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
	assertSystemdServicesRunningCounts(t, snap, 1, 2)
	assertSystemdServicesHasUnit(t, snap, "user", systemdFixtureUserUnit, systemdFixtureUserPID, systemdFixtureUserDesc)
	assertSystemdServicesHasUnit(t, snap, "system", systemdFixtureTailscaledUnit, systemdFixtureTailscaledPID, systemdFixtureTailscaledDesc)
	assertSystemdServicesHasUnit(t, snap, "system", systemdFixtureDockerUnit, systemdFixtureDockerPID, systemdFixtureDockerDesc)

	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```