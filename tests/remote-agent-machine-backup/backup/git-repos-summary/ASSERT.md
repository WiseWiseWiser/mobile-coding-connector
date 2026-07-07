## Expected Output

Dry-run summary includes `GIT REPOS` listing `.wrk-test/main` with branch `main`,
7-char short sha, `clean` status, and commit message on the next line. Stdout ends
with a trailing newline.

## Expected

1. Exit code 0.
2. Combined output contains `dry-run: machine backup plan`.
3. GIT REPOS section lists `.wrk-test/main`, `branch main`, `clean`, and `backup git fixture`.
4. Status line includes a 7-character hex short sha.
5. Stdout ends with `\n`.

## Side Effects

None (dry-run).

## Errors

- Missing GIT REPOS section or repo metadata lines.

## Exit Code

0.

```go
import (
	"regexp"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

var gitReposMainStatusRE = regexp.MustCompile(`(?m)^\s+branch main\s+[0-9a-f]{7}\s+clean\s*$`)

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
GIT REPOS:
    .wrk-test/main
      branch main  [0-9a-f]{7}  clean
      backup git fixture
`)

	assertGitReposSummaryContains(t, resp.Combined,
		".wrk-test/main",
		"branch main",
		"clean",
		gitFixtureCommitMsg,
	)
	section := gitReposSummarySection(resp.Combined)
	if !gitReposMainStatusRE.MatchString(section) {
		t.Fatalf("GIT REPOS missing branch main + 7-char sha + clean line; section:\n%s", section)
	}
	if !strings.Contains(resp.Stdout, gitFixtureCommitMsg) {
		t.Fatalf("stdout missing commit message; got:\n%s", resp.Stdout)
	}
	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```