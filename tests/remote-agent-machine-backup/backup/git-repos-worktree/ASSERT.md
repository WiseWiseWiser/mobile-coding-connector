## Expected Output

Dry-run summary lists main repo and linked worktree as flat table rows (KIND
`repo` / `worktree`) with branch `feature/foo`, 7-char sha, and `dirty (N modified)`
status on the worktree row. Stdout ends with a trailing newline.

## Expected

1. Exit code 0.
2. GIT REPOS table lists `.wrk-test/main` and `.wrk-test/feature-wt`.
3. Worktree row contains `worktree`, `feature/foo`, and `dirty (` with count.
4. Subheader reports `(1 repo, 1 worktree)`.
5. Stdout ends with `\n`.

## Side Effects

None (dry-run).

## Errors

- Missing worktree table row or dirty status count.

## Exit Code

0.

```go
import (
	"regexp"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

var (
	gitReposWorktreeCapturedAtRE = regexp.MustCompile(`captured_at: .+  \(1 repo, 1 worktree\)`)
	gitReposWorktreeDirtyRE      = regexp.MustCompile(`dirty \(\d+ modified\)`)
	gitReposWorktreeRowRE        = regexp.MustCompile(`worktree\s+\.wrk-test/feature-wt\s+feature/foo\s+[0-9a-f]{7}\s+dirty \(\d+ modified\)`)
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	section := gitReposSummarySection(resp.Combined)
	if section == "" {
		t.Fatalf("missing GIT REPOS summary section; got:\n%s", resp.Combined)
	}

	header := metaSectionHeaderLines(section, 1)
	assert.Output(t, header, `---
version: 2
---
GIT REPOS(.backup/git-repo-worktrees.json):
`)
	if !gitReposWorktreeCapturedAtRE.MatchString(section) {
		t.Fatalf("GIT REPOS missing repo/worktree count subheader; section:\n%s", section)
	}

	assertGitReposTableHeaders(t, section)
	assertGitReposSummaryContains(t, resp.Combined,
		".wrk-test/main",
		".wrk-test/feature-wt",
		"worktree",
		"feature/foo",
	)
	if !gitReposWorktreeRowRE.MatchString(section) {
		t.Fatalf("GIT REPOS missing worktree table row; section:\n%s", section)
	}
	if !gitReposWorktreeDirtyRE.MatchString(section) {
		t.Fatalf("GIT REPOS missing dirty status with modified count; section:\n%s", section)
	}
	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```