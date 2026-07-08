## Expected Output

Dry-run summary has GIT REPOS → INSTALLED → ENV → TOTAL with no SYSTEMD SERVICES line.
Archive omits `.backup/systemd-services.json`.

## Expected

1. Exit code 0.
2. Combined dry-run output does not contain `SYSTEMD SERVICES(.backup/systemd-services.json):`.
3. Meta sections before TOTAL: GIT REPOS → INSTALLED → ENV (no SYSTEMD SERVICES).
4. Archive does not list `.backup/systemd-services.json`.
5. Stdout ends with `\n`.

## Side Effects

Creates `systemd-services-meta-absent.tar.xz` under `agentHome`.

## Errors

- Unexpected SYSTEMD section or archive member without mock.

## Exit Code

0.

```go
import (
	"os"
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

	if strings.Contains(combined, "SYSTEMD SERVICES(.backup/systemd-services.json):") {
		t.Fatalf("unexpected SYSTEMD SERVICES section without mock; got:\n%s", combined)
	}
	if systemdServicesSummarySection(combined) != "" {
		t.Fatal("systemdServicesSummarySection non-empty without mock")
	}

	archiveHasXZMagic(t, resp.BackupPath)
	members := tarXZListMembers(t, resp.BackupPath)
	if memberListContains(members, ".backup/systemd-services.json") {
		t.Fatalf("archive unexpectedly contains systemd-services.json; members=%v", members)
	}

	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```