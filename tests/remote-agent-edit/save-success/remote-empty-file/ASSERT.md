## Expected Output

```
Downloading empty.md -> __STAGED__ (0 B)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (7 B, md5 __MD5__)
```

## Expected

1. Exit code 0 and a `Saved` line (the file already existed).
2. `Created` never appears.
3. Remote `empty.md` contains `filled\n`.

## Side Effects

- Remote `empty.md` replaced.

## Errors

- `Created` reported for an existing file.
- The empty base md5 treated as "no precondition", letting a concurrent change
  be overwritten.

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

	combinedHasAll(t, resp.Combined, "Saved "+resp.RemotePath)
	combinedHasNone(t, resp.Combined, "Created ", "is missing")
	assertRemoteContent(t, resp, "empty.md", "filled\n")

	assert.Output(t, resp.Stdout, `---
version: 2
__STAGED__: type=string
__EDITOR__: type=string
__REMOTE__: type=string
__MD5__: type=string
---
Downloading empty.md -> __STAGED__ (0 B)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (7 B, md5 __MD5__)
`)
}
```
