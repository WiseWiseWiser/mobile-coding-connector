## Expected Output

Dry-run summary has GIT REPOS → INSTALLED → ENV → TOTAL with no TAILSCALE line.
Archive omits `.backup/tailscale-config.json`.

## Expected

1. Exit code 0.
2. Combined dry-run output does not contain `TAILSCALE(.backup/tailscale-config.json):`.
3. Meta sections before TOTAL: GIT REPOS → INSTALLED → ENV (no TAILSCALE).
4. Archive does not list `.backup/tailscale-config.json`.
5. Stdout ends with `\n`.

## Side Effects

Creates `tailscale-meta-absent.tar.xz` under `agentHome`.

## Errors

- Unexpected TAILSCALE section or archive member without mock.

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

	if strings.Contains(combined, "TAILSCALE(.backup/tailscale-config.json):") {
		t.Fatalf("unexpected TAILSCALE section without mock; got:\n%s", combined)
	}
	if tailscaleSummarySection(combined) != "" {
		t.Fatal("tailscaleSummarySection non-empty without mock")
	}

	archiveHasXZMagic(t, resp.BackupPath)
	members := tarXZListMembers(t, resp.BackupPath)
	if memberListContains(members, ".backup/tailscale-config.json") {
		t.Fatalf("archive unexpectedly contains tailscale-config.json; members=%v", members)
	}

	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```