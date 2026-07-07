## Expected Output

Archive omits seeded paths under five built-in path-prefix exclusions and includes
`.config/confluence-fetch-skill/data` (reverted exclusion).

## Expected

1. Exit code 0.
2. `OutputPath` exists with xz magic bytes.
3. Archive member list omits:
   - `.codex/.tmp/junk`
   - `.local/share/opencode/repos/foo/clone`
   - `.local/share/cursor-agent/versions/v1/pkg`
   - `.opencode/bin/opencode`
4. Archive member list includes `.config/confluence-fetch-skill/data/cache`.

## Side Effects

Creates `path-exclusions-backup.tar.xz` under `agentHome`.

## Errors

- Any path-prefix excluded member present in archive listing.

## Exit Code

0.

```go
import (
	"os"
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

	archiveHasXZMagic(t, resp.BackupPath)
	members := tarXZListMembers(t, resp.BackupPath)
	for _, absent := range []string{
		".codex/.tmp/junk",
		".local/share/opencode/repos/foo/clone",
		".local/share/cursor-agent/versions/v1/pkg",
		".opencode/bin/opencode",
	} {
		if memberListContains(members, absent) {
			t.Fatalf("archive unexpectedly contains path-excluded %q", absent)
		}
	}
	if !memberListContains(members, ".config/confluence-fetch-skill/data/cache") {
		t.Fatalf("archive missing reverted inclusion %q; members=%v", ".config/confluence-fetch-skill/data/cache", members)
	}
}
```