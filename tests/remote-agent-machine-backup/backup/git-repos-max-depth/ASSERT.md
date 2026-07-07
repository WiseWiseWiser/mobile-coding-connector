## Expected Output

Deep repo beyond `--git-dirs-scan-max-depth 2` is not discovered; summary prints
`GIT REPOS: (none)`. Stdout ends with a trailing newline.

## Expected

1. Exit code 0.
2. Combined output contains `GIT REPOS: (none)`.
3. GIT REPOS section does not list `deep-repo`.
4. Stdout ends with `\n`.

## Side Effects

None (dry-run).

## Errors

- Deep repo listed despite max depth cap.

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
	if strings.Contains(section, "deep-repo") {
		t.Fatalf("GIT REPOS unexpectedly lists deep repo; section:\n%s", section)
	}
	assertStdoutEndsWithNewline(t, resp.Stdout)
}
```