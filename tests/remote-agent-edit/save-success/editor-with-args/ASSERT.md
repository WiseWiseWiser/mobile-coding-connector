## Expected Output

```
Downloading notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ --mark __STAGED__
Saved __REMOTE__ (10 B, md5 __MD5__)
```

## Expected

1. Exit code 0.
2. The `Opening` line shows the editor command including its argument.
3. The editor still receives the staged path as its last argument.
4. Remote `notes.md` contains `with args\n`.

## Side Effects

- Remote `notes.md` replaced.

## Errors

- Editor invoked without the extra argument (flag value not split).
- Staged path passed as the wrong argv element (editor edits nothing, `file not changed`).

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

	if !strings.Contains(resp.Stdout, "Opening "+resp.EditorPath+" "+resp.StagedPath) {
		t.Fatalf("Opening line missing editor args;\nhave:\n%s", resp.Stdout)
	}
	combinedHasAll(t, resp.Combined, "Saved "+resp.RemotePath)
	assertRemoteContent(t, resp, "notes.md", "with args\n")

	assert.Output(t, resp.Stdout, `---
version: 2
__EDITOR__: type=string
__STAGED__: type=string
__REMOTE__: type=string
__MD5__: type=string
---
Downloading notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ --mark __STAGED__
Saved __REMOTE__ (10 B, md5 __MD5__)
`)
}
```
