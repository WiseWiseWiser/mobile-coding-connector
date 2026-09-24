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
  current  md5: __THEIRSMD5__
  your edits are kept at: __STAGED__
hint: reconcile, then push your copy:
        remote-agent download __REMOTE__ __STAGED__.remote
        remote-agent upload __STAGED__ __REMOTE__
```

## Expected

1. Exit code 1.
2. The conflict block names the remote path, both md5s, and the staged copy.
3. The hint lists the two resolution commands (`download` then `upload`).
4. Remote `notes.md` still contains `theirs\n`: the CLI wrote nothing.
5. The staged copy still contains `mine\n` (the user's edits survive).

## Side Effects

- None on the server; the staged copy is kept for manual reconciliation.

## Errors

- Exit 0 / `Saved` (precondition bypassed → data loss).
- Staged copy removed or overwritten, losing the user's edits.

## Exit Code

1.

```go
import (
	"strings"
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
		"current  md5: "+md5Hex("theirs\n"),
		"your edits are kept at: "+resp.StagedPath,
		"remote-agent download "+resp.RemotePath+" "+resp.StagedPath+".remote",
		"remote-agent upload "+resp.StagedPath+" "+resp.RemotePath,
	)
	combinedHasNone(t, resp.Combined, "Saved ", "Created ")

	if !strings.Contains(resp.Stderr, "changed on the server") {
		t.Fatalf("conflict block should be on stderr;\nstderr:\n%s", resp.Stderr)
	}
	if strings.Contains(resp.Stdout, "changed on the server") {
		t.Fatalf("conflict block must not be on stdout;\nstdout:\n%s", resp.Stdout)
	}

	assertRemoteContent(t, resp, "notes.md", "theirs\n")
	assertStagedContent(t, resp, "mine\n")
}
```
