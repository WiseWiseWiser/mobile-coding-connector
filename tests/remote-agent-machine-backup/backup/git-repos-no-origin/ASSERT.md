## Expected Output

Dry-run summary lists `.wrk-test/main` in a GIT REPOS table row with branch `main`,
7-char short sha, `clean` status, commit subject, and ORIGIN `(none)`. Stdout ends
with a trailing newline.

## Expected

1. Exit code 0.
2. GIT REPOS table lists main repo metadata and ORIGIN `(none)`.
3. Stdout ends with `\n`.

## Side Effects

None (dry-run).

## Errors

- Missing `(none)` ORIGIN cell or unexpected origin URL in summary.

## Exit Code

0.

```go
import (
	"regexp"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

var (
	gitReposCapturedAtRE   = regexp.MustCompile(`captured_at: .+  \(1 repo, 0 worktree\)`)
	gitReposNoOriginRowRE = regexp.MustCompile(`repo\s+\.wrk-test/main\s+main\s+[0-9a-f]{7}\s+clean\s+\(none\)\s+` + regexp.QuoteMeta(gitFixtureCommitMsg))
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
		".wrk-test/main",
		"repo",
		"main",
		"clean",
		gitFixtureCommitMsg,
		"(none)",
	)
	if !gitReposNoOriginRowRE.MatchString(section) {
		t.Fatalf("GIT REPOS missing repo row with (none) origin; section:\n%s", section)
	}
	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```