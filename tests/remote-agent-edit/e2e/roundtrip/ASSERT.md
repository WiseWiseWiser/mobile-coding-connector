---
label: heavy, e2e
---

## Expected Output

```
Downloading notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (4 B, md5 __MD5__)
```

## Expected

1. Exit code 0 from the product binary.
2. `Saved` line names the resolved remote path and the new md5.
3. Remote `notes.md` contains `e2e\n`.
4. The server (started with `HOME=serverHome`) accepted the conditional write.

## Side Effects

- Session cache holds the built binaries; temp homes are removed after the leaf.

## Errors

- Config-file resolution or bearer auth broken in the product binary path.
- Conditional write rejected on a real server.

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
		"Downloading notes.md -> "+resp.StagedPath+" (4 B)",
		"Saved "+resp.RemotePath,
		md5Hex("e2e\n"),
	)
	assertRemoteContent(t, resp, "notes.md", "e2e\n")
	assertStagedContent(t, resp, "e2e\n")

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
