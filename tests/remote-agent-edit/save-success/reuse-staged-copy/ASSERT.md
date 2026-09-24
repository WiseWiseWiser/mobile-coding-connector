## Expected Output

```
Skipped download: __STAGED__ already matches the remote (md5 __MD5__)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (4 B, md5 __MD5B__)
```

## Expected

1. Exit code 0 and the `Skipped download: … already matches the remote (md5 …)` meta line.
2. No `Downloading` line and **zero** `/api/files/download` requests.
3. The check request still happens (digest lookup) and the save is written normally.
4. Remote `notes.md` contains `new\n`; the staged copy contains `new\n`.

## Side Effects

- Remote `notes.md` replaced; no bytes transferred from the server to the client.

## Errors

- `Downloading` line / one download request (optimization not applied).
- A conflict instead of `Saved` (stale digest used as the base).

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

	combinedHasAll(t, resp.Combined,
		"Skipped download: "+resp.StagedPath+" already matches the remote (md5 "+md5Hex(standardRemoteInitial)+")",
		"Saved "+resp.RemotePath,
	)
	combinedHasNone(t, resp.Combined, "Downloading", "changed on the server", "Error")

	assertRequestCount(t, resp, "/api/files/download", 0)
	assertRequestCount(t, resp, "/api/files/check", 1)
	assertRequestCount(t, resp, "/api/files/write", 1)

	assertRemoteContent(t, resp, "notes.md", "new\n")
	assertStagedContent(t, resp, "new\n")

	assert.Output(t, resp.Stdout, `---
version: 2
__STAGED__: type=string
__EDITOR__: type=string
__REMOTE__: type=string
__MD5__: type=string
__MD5B__: type=string
---
Skipped download: __STAGED__ already matches the remote (md5 __MD5__)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (4 B, md5 __MD5B__)
`)
}
```
