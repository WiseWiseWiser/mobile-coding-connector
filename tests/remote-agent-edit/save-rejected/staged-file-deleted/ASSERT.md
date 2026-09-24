## Expected Output

stdout:

```
Downloading notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
```

stderr:

```
Error: staged copy __STAGED__ is gone; nothing was uploaded
```

## Expected

1. Exit code 1.
2. The error names the missing staged path.
3. No `Saved`/`Created`/`file not changed` line.
4. Remote `notes.md` still contains `old\n` (the remote file is not deleted).

## Side Effects

- None on the server.

## Errors

- Remote file deleted because the staged copy disappeared.
- `file not changed` reported for a missing staged file.

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
		"Error: staged copy "+resp.StagedPath+" is gone; nothing was uploaded",
	)
	combinedHasNone(t, resp.Combined, "Saved ", "Created ", "file not changed")

	assertRemoteContent(t, resp, "notes.md", "old\n")
	assertRemoteMissing(t, resp, "notes.md.bak")
}
```
