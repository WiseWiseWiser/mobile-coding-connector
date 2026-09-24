## Expected Output

```
Downloading notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (4 B, md5 __MD5__)
```

## Expected

1. Exit code 0 and a `Saved` line.
2. No conflict, even though the staged path already existed with the same size:
   the fresh download replaced it before the editor ran.
3. Remote `notes.md` contains `new\n`; the staged copy contains `new\n` (not `ZZZZ`).

## Side Effects

- Stale staged copy replaced by the current remote content, then by the edit.

## Errors

- `Error: ... changed on the server ...` (409) — the stale same-size copy was
  used as the base version (resume/skip bug).
- Staged content still `ZZZZ`.

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

	combinedHasAll(t, resp.Combined, "Saved "+resp.RemotePath)
	combinedHasNone(t, resp.Combined, "changed on the server", "Error")

	assertRemoteContent(t, resp, "notes.md", "new\n")
	assertStagedContent(t, resp, "new\n")

	assert.Output(t, resp.Stdout, `---
version: 2
__STAGED__: type=string
__EDITOR__: type=string
__REMOTE__: type=string
__MD5__: type=string
---
Downloading notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (4 B, md5 __MD5__)
`)
}
```
