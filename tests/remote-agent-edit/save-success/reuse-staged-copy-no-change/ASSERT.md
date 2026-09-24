## Expected Output

```
Skipped download: __STAGED__ already matches the remote (md5 __MD5__)
Opening __EDITOR__ __STAGED__
file not changed
```

## Expected

1. Exit code 0, `Skipped download`, and the trailing `file not changed` line.
2. Zero `/api/files/download` and zero `/api/files/write` requests.
3. Remote `notes.md` still contains `old\n`.

## Side Effects

- None beyond the local hash; no transfer and no write.

## Errors

- Any download or write request for an unchanged file.

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
	combinedHasNone(t, resp.Combined, "Downloading", "Saved ", "Error")

	assertRequestCount(t, resp, "/api/files/download", 0)
	assertRequestCount(t, resp, "/api/files/write", 0)
	assertRemoteContent(t, resp, "notes.md", "old\n")

	assert.Output(t, resp.Stdout, `---
version: 2
__STAGED__: type=string
__EDITOR__: type=string
__MD5__: type=string
---
Skipped download: __STAGED__ already matches the remote (md5 __MD5__)
Opening __EDITOR__ __STAGED__
file not changed
`)
}
```
