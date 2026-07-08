## Expected Output

Dry-run summary prints `GIT REPOS(.backup/git-repo-worktrees.json): (skipped)`.
Real backup archive omits `.backup/git-repo-worktrees.json`. Stdout ends with a
trailing newline.

## Expected

1. Exit code 0.
2. Dry-run combined output contains `GIT REPOS(.backup/git-repo-worktrees.json): (skipped)`.
3. Archive member list does not contain `.backup/git-repo-worktrees.json`.
4. Stdout ends with `\n`.

## Side Effects

Creates `git-repos-skipped.tar.xz` under `agentHome`.

## Errors

- Scan ran despite skip flag, or git JSON leaked into archive.

## Exit Code

0.

```go
import (
	"os"
	"testing"

	"github.com/xhd2015/doctest/assert"
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

	assert.Output(t, gitReposSummarySection(resp.DryRunCombined), `---
version: 2
---
GIT REPOS(.backup/git-repo-worktrees.json): (skipped)
`)

	members := tarXZListMembers(t, resp.BackupPath)
	if memberListContains(members, ".backup/git-repo-worktrees.json") {
		t.Fatalf("archive unexpectedly contains git-repo-worktrees.json; members=%v", members)
	}
	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```