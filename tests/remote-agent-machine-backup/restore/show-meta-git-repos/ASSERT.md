## Expected Output

`restore --show-meta` prints `=== .backup/git-repo-worktrees.json ===` with archive
JSON content including `captured_at` and `.wrk-test/main` repo entry. Stdout ends
with a trailing newline.

## Expected

1. Exit code 0.
2. Combined output contains `=== .backup/git-repo-worktrees.json ===`.
3. Combined output contains `captured_at` and `.wrk-test/main`.
4. Archive git JSON parses with version `1.0` and 7-char repo sha.
5. Stdout ends with `\n`.

## Side Effects

None (read-only archive inspection).

## Errors

- Missing git meta section or invalid archive JSON.

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
		"=== .backup/git-repo-worktrees.json ===",
		"captured_at",
		".wrk-test/main",
	)

	raw := tarXZExtractFile(t, resp.BackupPath, ".backup/git-repo-worktrees.json")
	snap := parseGitRepoWorktreesJSON(t, raw)
	assertGitRepoSnapshotBasics(t, snap, ".wrk-test/main")
	if !strings.Contains(resp.Combined, snap.CapturedAt) {
		t.Fatalf("stdout missing captured_at %q; got:\n%s", snap.CapturedAt, resp.Combined)
	}
	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```