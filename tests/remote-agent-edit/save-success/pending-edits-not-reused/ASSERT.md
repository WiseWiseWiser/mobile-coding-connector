## Expected Output

```
Downloading notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
file not changed
```

## Expected

1. Exit code 0 with a `Downloading` line and no `Skipped download` line.
2. Exactly one `/api/files/download` request.
3. The staged copy holds `old\n` — the download replaced the divergent local bytes
   (a no-op editor proves it was not the download that wrote `mine\n`).
4. Remote `notes.md` is untouched.

## Side Effects

- Pending staged content replaced by the current remote content (documented
  re-run behavior), then reported as unchanged.

## Errors

- `Skipped download` for divergent content, leaving the user editing stale bytes
  while the base md5 claims otherwise.

## Exit Code

0.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"

	"github.com/xhd2015/doctest/assert"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	assertExit(t, resp, 0)

	combinedHasAll(t, resp.Combined, "Downloading", "file not changed")
	combinedHasNone(t, resp.Combined, "Skipped download", "Saved ", "Error")

	assertRequestCount(t, resp, "/api/files/download", 1)
	assertStagedContent(t, resp, "old\n")
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
