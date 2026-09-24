## Expected Output

stdout:

```
Downloading notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (4 B, md5 __MD5__)
```

stderr:

```
warning: server did not report the remote digest; downloading the full copy
```

## Expected

1. Exit code 0 — a missing digest never fails the edit.
2. The warning appears on stderr, not stdout.
3. A `Downloading` line and exactly one `/api/files/download` request; no
   `Skipped download` line.
4. Remote `notes.md` contains `new\n`.

## Side Effects

- Identical staged copy re-downloaded (no optimization available).

## Errors

- Exit 1 with a digest error on an old server.
- `Skipped download` claimed without a digest.

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

	combinedHasAll(t, resp.Combined,
		"warning: server did not report the remote digest; downloading the full copy",
		"Downloading",
		"Saved "+resp.RemotePath,
	)
	combinedHasNone(t, resp.Combined, "Skipped download", "Error")

	if !strings.Contains(resp.Stderr, "did not report the remote digest") {
		t.Fatalf("warning should be on stderr;\nstderr:\n%s", resp.Stderr)
	}
	if strings.Contains(resp.Stdout, "did not report the remote digest") {
		t.Fatalf("warning must not be on stdout;\nstdout:\n%s", resp.Stdout)
	}
	assertRequestCount(t, resp, "/api/files/download", 1)
	assertRemoteContent(t, resp, "notes.md", "new\n")

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
