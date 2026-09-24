## Expected Output

stdout:

```
Downloading notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
```

stderr:

```
Error: editor exited with status 3; nothing was uploaded (staged copy: __STAGED__)
```

## Expected

1. Exit code 1.
2. The error names the editor exit status and the staged copy path.
3. Remote `notes.md` still contains `old\n` — no write request was made.
4. The staged copy still holds the editor's content (nothing was reverted).

## Side Effects

- None on the server.

## Errors

- The aborted edit is uploaded anyway.
- Error message hides where the staged copy lives.

## Exit Code

1.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	assertExit(t, resp, 1)

	combinedHasAll(t, resp.Combined,
		"Error: editor exited with status 3; nothing was uploaded (staged copy: "+resp.StagedPath+")",
	)
	combinedHasNone(t, resp.Combined, "Saved ", "Created ", "file not changed")

	assertRemoteContent(t, resp, "notes.md", "old\n")
	assertStagedContent(t, resp, "should not be saved\n")
}
```
