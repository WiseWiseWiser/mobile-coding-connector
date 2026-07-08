## Expected Output

Stdout prints `=== .backup/systemd-services.json ===` with archive content including
`captured_at`, `"systemd_available": true`, and mock running service units. Does not
print `config.json`.

## Expected

1. Exit code 0.
2. Combined output contains `=== .backup/systemd-services.json ===` and `captured_at`.
3. Combined output contains `"systemd_available": true` and fixture unit names.
4. Combined output does not contain `=== .backup/config.json ===`.

## Side Effects

None (read-only archive inspection).

## Errors

- Missing systemd meta section.

## Exit Code

0.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if resp.BackupPath == "" {
		t.Fatal("prereq BackupPath empty")
	}

	combinedHasAll(t, resp.Combined,
		"=== .backup/systemd-services.json ===",
		"captured_at",
		`"systemd_available": true`,
		systemdFixtureUserUnit,
		systemdFixtureTailscaledUnit,
		systemdFixtureDockerUnit,
	)
	combinedHasNone(t, resp.Combined, "=== .backup/config.json ===")

	raw := tarXZExtractFile(t, resp.BackupPath, ".backup/systemd-services.json")
	snap := parseSystemdServicesJSON(t, raw)
	assertSystemdServicesBasics(t, snap)
	assertSystemdServicesRunningCounts(t, snap, 1, 2)
	if !strings.Contains(resp.Combined, snap.CapturedAt) {
		t.Fatalf("stdout missing captured_at %q; got:\n%s", snap.CapturedAt, resp.Combined)
	}
}
```