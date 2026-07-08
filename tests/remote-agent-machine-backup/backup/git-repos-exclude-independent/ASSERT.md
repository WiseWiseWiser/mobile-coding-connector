## Expected Output

Dry-run summary includes `GIT REPOS(.backup/git-repo-worktrees.json):` with a flat
table listing `.wrk-test/main` (branch `main`, 7-char short sha, `clean`, commit
subject). `.wrk-test` is excluded from the backup plan (not listed under DOT DIRS).
Stdout ends with a trailing newline.

## Expected

1. Exit code 0.
2. Combined output contains `dry-run: machine backup plan`.
3. GIT REPOS table lists `.wrk-test/main`, `main`, `clean`, and `backup git fixture`.
4. Status includes a 7-character hex short sha.
5. DOT DIRS summary does not include `.wrk-test` (excluded from backup).
6. Stdout ends with `\n`.

## Side Effects

None (dry-run).

## Errors

- GIT REPOS omits `.wrk-test/main` when exclude is set.
- `.wrk-test` appears as an included dot-dir.

## Exit Code

0.

```go
import (
	"regexp"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

var (
	gitReposCapturedAtRE        = regexp.MustCompile(`captured_at: .+  \(1 repo, 0 worktree\)`)
	gitReposExcludeIndepRowRE   = regexp.MustCompile(`repo\s+\.wrk-test/main\s+main\s+[0-9a-f]{7}\s+clean\s+.*` + regexp.QuoteMeta(gitFixtureCommitMsg))
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
	)
	if !gitReposExcludeIndepRowRE.MatchString(section) {
		t.Fatalf("GIT REPOS missing repo table row; section:\n%s", section)
	}
	if !strings.Contains(resp.Stdout, gitFixtureCommitMsg) {
		t.Fatalf("stdout missing commit message; got:\n%s", resp.Stdout)
	}
	assertDotDirsExcludes(t, resp.Combined, ".wrk-test")
	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```