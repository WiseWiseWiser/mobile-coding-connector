## Expected Output

```
Downloading notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
file not changed
```

## Expected

1. Exit code 0 and the trailing `file not changed` line.
2. No `Saved`/`Created` line and no conflict.
3. Remote `notes.md` still contains `old\n`.

## Side Effects

- None on the server (no write request is issued).

## Errors

- A write is issued for identical content (wasted request, possible conflict
  against a concurrent change).

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
