## Expected Output

Dry-run completes with exit 0. GIT REPOS lists `.wrk-test/empty` with an `error:`
line for the unborn HEAD (no commits). Stdout ends with a trailing newline.

## Expected

1. Exit code 0.
2. Combined output contains `dry-run: machine backup plan`.
3. GIT REPOS section lists `.wrk-test/empty` with `error: no commits (HEAD unborn)`.
4. Stdout ends with `\n`.

## Side Effects

None (dry-run).

## Errors

- Non-zero exit or missing durable error line for empty repo.

## Exit Code

0.

```go
import (
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

	assert.Output(t, gitReposSummarySection(resp.Combined), `---
version: 2
---
GIT REPOS:
    .wrk-test/empty
      error: no commits (HEAD unborn)
`)

	assertGitReposSummaryContains(t, resp.Combined,
		".wrk-test/empty",
		"error: no commits (HEAD unborn)",
	)
	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```