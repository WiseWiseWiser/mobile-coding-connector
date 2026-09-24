## Expected Output

stdout:

```
Downloading notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
```

stderr:

```
Error: __REMOTE__ changed on the server while you were editing; nothing was written
  downloaded md5: __BASEMD5__
  current  md5: __EMPTYMD5__
  the remote file was deleted
  your edits are kept at: __STAGED__
hint: reconcile, then push your copy:
        remote-agent download __REMOTE__ __STAGED__.remote
        remote-agent upload __STAGED__ __REMOTE__
```

## Expected

1. Exit code 1.
2. The conflict block reports the empty-content md5
   (`d41d8cd98f00b204e9800998ecf8427e`) and the deletion line.
3. The remote file is still missing (not recreated from the staged copy).
4. The staged copy keeps `mine\n` so the user can decide.

## Side Effects

- None on the server.

## Errors

- The deleted file is silently recreated (`Created`/`Saved`).
- Conflict reported without explaining that the remote file is gone.

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
		"Error: "+resp.RemotePath+" changed on the server while you were editing; nothing was written",
		"downloaded md5: "+md5Hex("old\n"),
		"current  md5: d41d8cd98f00b204e9800998ecf8427e",
		"the remote file was deleted",
		"your edits are kept at: "+resp.StagedPath,
	)
	combinedHasNone(t, resp.Combined, "Saved ", "Created ")

	assertRemoteMissing(t, resp, "notes.md")
	assertStagedContent(t, resp, "mine\n")
}
```
