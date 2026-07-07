## Expected Output

Dry-run summary prints `GIT REPOS: (none)` when no repositories are found.
Stdout ends with a trailing newline.

## Expected

1. Exit code 0.
2. Combined output contains `GIT REPOS: (none)`.
3. Combined output does not list repo paths under GIT REPOS.
4. Stdout ends with `\n`.

## Side Effects

None (dry-run).

## Errors

- Missing `(none)` marker or unexpected repo listings.

## Exit Code

0.

```go
import (
	"strings"
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
GIT REPOS: (none)
`)

	section := gitReposSummarySection(resp.Combined)
	if strings.Contains(section, ".wrk-test") {
		t.Fatalf("GIT REPOS unexpectedly lists repo paths; section:\n%s", section)
	}
	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```