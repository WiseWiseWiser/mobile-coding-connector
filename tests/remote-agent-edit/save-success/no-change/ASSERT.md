## Expected Output

```
Downloading notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
file not changed
```

## Expected

1. Exit code 0.
2. Stdout ends with exactly `file not changed`.
3. Remote `notes.md` still contains `old\n` and no write request was made.
4. The staged copy still holds the downloaded content.

## Side Effects

- None beyond the staged copy under the staging dir.

## Errors

- `Saved` printed for an unchanged file, or a conflict because the unchanged
  content was still uploaded.

## Exit Code

0.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"

	"github.com/xhd2015/doctest/assert"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	assertExit(t, resp, 0)

	if !strings.HasSuffix(resp.Stdout, "file not changed\n") {
		t.Fatalf("stdout should end with %q;\nhave:\n%s", "file not changed", resp.Stdout)
	}
	combinedHasNone(t, resp.Combined, "Saved ", "Created ", "Error")

	assertRemoteContent(t, resp, "notes.md", "old\n")
	assertStagedContent(t, resp, "old\n")

	assert.Output(t, resp.Stdout, `---
version: 2
__STAGED__: type=string
__EDITOR__: type=string
---
Downloading notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
file not changed
`)
}
```
