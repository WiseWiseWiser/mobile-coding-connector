## Expected Output

Dry-run succeeds; backup archive members (excluding `manifest.json` and `.backup/`
meta) equal the server plan `included` path set.

## Expected

1. Dry-run exit code 0 with `dry-run: machine backup plan` in `DryRunCombined`.
2. Backup exit code 0; `BackupPath` exists with xz magic.
3. `DryRunIncluded` set equals archive user members (sorted).
4. Archive contains `manifest.json` and injected `.backup/` meta (not in included set).

## Side Effects

Creates `dry-run-matches-archive.tar.xz` under `agentHome`.

## Errors

- Dry-run or backup non-zero exit.
- Included set / archive member mismatch.
- Missing archive.

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
		t.Fatalf("backup exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if !strings.Contains(resp.DryRunCombined, "dry-run: machine backup plan") {
		t.Fatalf("dry-run missing plan summary; got:\n%s", resp.DryRunCombined)
	}
	if resp.BackupPath == "" {
		t.Fatal("BackupPath empty")
	}
	if _, err := os.Stat(resp.BackupPath); err != nil {
		t.Fatalf("backup file missing: %v", err)
	}
	if len(resp.DryRunIncluded) == 0 {
		t.Fatal("DryRunIncluded empty")
	}

	archiveHasXZMagic(t, resp.BackupPath)
	members := tarXZListMembers(t, resp.BackupPath)
	archiveUsers := archiveUserMembers(members)
	assertStringSetsEqual(t, "dry-run included vs archive", resp.DryRunIncluded, archiveUsers)

	if !memberListContains(members, "manifest.json") {
		t.Fatalf("archive missing manifest.json; members=%v", members)
	}
	hasBackupMeta := false
	for _, m := range members {
		if strings.HasPrefix(strings.TrimPrefix(m, "./"), ".backup/") {
			hasBackupMeta = true
			break
		}
	}
	if !hasBackupMeta {
		t.Fatalf("archive missing injected .backup/ meta; members=%v", members)
	}
}
```