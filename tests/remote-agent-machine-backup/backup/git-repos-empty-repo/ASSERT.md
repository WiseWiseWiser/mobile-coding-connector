## Expected Output

Dry-run completes with exit 0. GIT REPOS table lists `.wrk-test/empty` with
STATUS `error: no commits (HEAD unborn)` (PATH filled; other columns blank).
Stdout ends with a trailing newline.

## Expected

1. Exit code 0.
2. Combined output contains `dry-run: machine backup plan`.
3. GIT REPOS section lists `.wrk-test/empty` with `error: no commits (HEAD unborn)`.
4. Stdout ends with `\n`.

## Side Effects

None (dry-run).

## Errors

- Non-zero exit or missing durable error row for empty repo.

## Exit Code

0.

```go
import (
	"regexp"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

var (
	gitReposCapturedAtRE      = regexp.MustCompile(`captured_at: .+  \(1 repo, 0 worktree\)`)
	gitReposEmptyErrorRowRE   = regexp.MustCompile(`(?m)^\s+repo\s+\.wrk-test/empty(?:\s+){2,}error: no commits \(HEAD unborn\)\s*$`)
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
	if !gitReposCapturedAtRE.MatchString(section) {
		t.Fatalf("GIT REPOS missing repo/worktree count subheader; section:\n%s", section)
	}

	assertGitReposTableHeaders(t, section)
	assertGitReposSummaryContains(t, resp.Combined,
		".wrk-test/empty",
		"error: no commits (HEAD unborn)",
	)
	if !gitReposEmptyErrorRowRE.MatchString(section) {
		t.Fatalf("GIT REPOS missing error-only table row; section:\n%s", section)
	}
	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```