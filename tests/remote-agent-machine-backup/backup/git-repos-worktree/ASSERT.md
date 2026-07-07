## Expected Output

Dry-run summary nests a `worktree .wrk-test/feature-wt` block with branch
`feature/foo`, 7-char sha, and `dirty (1 modified)` status. Stdout ends with a
trailing newline.

## Expected

1. Exit code 0.
2. GIT REPOS section lists main repo `.wrk-test/main` and worktree `.wrk-test/feature-wt`.
3. Worktree block contains `worktree`, `branch feature/foo`, and `dirty (` with count.
4. Stdout ends with `\n`.

## Side Effects

None (dry-run).

## Errors

- Missing worktree nesting or dirty status count.

## Exit Code

0.

```go
import (
	"regexp"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

var gitReposWorktreeDirtyRE = regexp.MustCompile(`dirty \(\d+ modified\)`)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	assert.Output(t, gitReposSummarySection(resp.Combined), `---
version: 2
---
...4 lines omitted...
      worktree .wrk-test/feature-wt
        branch feature/foo  [0-9a-f]{7}  dirty \(\d+ modified\)
...1 lines omitted...
`)

	assertGitReposSummaryContains(t, resp.Combined,
		".wrk-test/main",
		"worktree .wrk-test/feature-wt",
		"branch feature/foo",
	)
	section := gitReposSummarySection(resp.Combined)
	if !strings.Contains(section, "worktree") {
		t.Fatalf("GIT REPOS missing worktree prefix; section:\n%s", section)
	}
	if !gitReposWorktreeDirtyRE.MatchString(section) {
		t.Fatalf("GIT REPOS missing dirty status with modified count; section:\n%s", section)
	}
	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```