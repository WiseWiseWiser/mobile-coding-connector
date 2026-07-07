## Expected Output

Real backup archive contains `.backup/git-repo-worktrees.json` with version `1.0`,
`captured_at`, repo path `.wrk-test/main`, 7-char sha, and status. Stdout ends
with a trailing newline.

## Expected

1. Exit code 0.
2. Archive lists `.backup/git-repo-worktrees.json` member.
3. Extracted JSON has `version` `1.0`, non-empty `captured_at`, and repo `.wrk-test/main`.
4. Repo `commit_sha` is 7-character hex; `status` is non-empty.
5. Stdout ends with `\n`.

## Side Effects

Creates `git-repos-archive.tar.xz` under `agentHome`.

## Errors

- Missing or invalid git-repo-worktrees.json in archive.

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

	archiveHasXZMagic(t, resp.BackupPath)
	members := tarXZListMembers(t, resp.BackupPath)
	if !memberListContains(members, ".backup/git-repo-worktrees.json") {
		t.Fatalf("archive missing .backup/git-repo-worktrees.json; members=%v", members)
	}

	raw := tarXZExtractFile(t, resp.BackupPath, ".backup/git-repo-worktrees.json")
	snap := parseGitRepoWorktreesJSON(t, raw)
	assertGitRepoSnapshotBasics(t, snap, ".wrk-test/main")

	for _, repo := range snap.Repos {
		if repo.Path != ".wrk-test/main" {
			continue
		}
		if repo.Branch != "main" {
			t.Fatalf("repo branch = %q, want main", repo.Branch)
		}
		if !strings.Contains(repo.CommitMsg, gitFixtureCommitMsg) {
			t.Fatalf("repo commit_msg = %q, want containing %q", repo.CommitMsg, gitFixtureCommitMsg)
		}
		if repo.Status != "clean" {
			t.Fatalf("repo status = %q, want clean", repo.Status)
		}
	}
	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```