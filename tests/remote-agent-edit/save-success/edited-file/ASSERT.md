## Expected Output

```
Downloading notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (4 B, md5 __MD5__)
```

## Expected

1. Exit code 0.
2. Stdout shows the download line, the editor line, and a `Saved` line with size and md5.
3. Remote `notes.md` contains `new\n`.
4. The staged copy is kept under the staging dir with the same content.

## Side Effects

- Remote `notes.md` replaced (mode preserved); staged copy left in place.

## Errors

- Save rejected although nobody else touched the file (stale base / resume bug).
- `Saved` reports a path other than the resolved remote path.

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

	assertStdoutEndsWithNewline(t, resp.Stdout)
	combinedHasAll(t, resp.Combined,
		"Downloading notes.md -> "+resp.StagedPath+" (4 B)",
		"Opening ",
		resp.StagedPath,
		"Saved "+resp.RemotePath,
		md5Hex("new\n"),
	)
	combinedHasNone(t, resp.Combined, "file not changed", "warning:")

	assertRemoteContent(t, resp, "notes.md", "new\n")
	assertStagedContent(t, resp, "new\n")
	assertStagedPathUnderStaging(t, resp)
	assertStagedMode(t, resp, 0600)

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
