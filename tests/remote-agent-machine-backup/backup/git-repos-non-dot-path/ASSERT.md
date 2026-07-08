## Expected Output

Dry-run summary includes `GIT REPOS(.backup/git-repo-worktrees.json):` with a flat
table listing `projects/demo` (KIND `repo`, branch `main`, 7-char short sha, `clean`
status, full commit subject). Stdout ends with a trailing newline.

## Expected

1. Exit code 0.
2. Combined output contains `dry-run: machine backup plan`.
3. GIT REPOS table lists `projects/demo`, `main`, `clean`, and `non-dot fixture`.
4. Status includes a 7-character hex short sha.
5. Stdout ends with `\n`.

## Side Effects

None (dry-run).

## Errors

- Missing GIT REPOS section or repo metadata table row.

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
	gitReposCapturedAtRE = regexp.MustCompile(`captured_at: .+  \(1 repo, 0 worktree\)`)
	gitReposNonDotRowRE  = regexp.MustCompile(`repo\s+projects/demo\s+main\s+[0-9a-f]{7}\s+clean\s+.*` + regexp.QuoteMeta(gitNonDotFixtureCommitMsg))
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
		"projects/demo",
		"repo",
		"main",
		"clean",
		gitNonDotFixtureCommitMsg,
	)
	if !gitReposNonDotRowRE.MatchString(section) {
		t.Fatalf("GIT REPOS missing repo table row; section:\n%s", section)
	}
	if !strings.Contains(resp.Stdout, gitNonDotFixtureCommitMsg) {
		t.Fatalf("stdout missing commit message; got:\n%s", resp.Stdout)
	}
	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```