## Expected Output

```
Remote notes/todo.md is missing; starting from an empty file
Opening __EDITOR__ __STAGED__
Created __REMOTE__ (9 B, md5 __MD5__)
```

## Expected

1. Exit code 0.
2. Stdout reports the missing remote file and a `Created` line.
3. Remote `notes/todo.md` exists with `buy milk\n` (parent directory created).
4. No `Downloading` line (nothing was downloaded).

## Side Effects

- Remote `notes/todo.md` created (0644) with its parent directory.

## Errors

- Download attempted for a missing file, or `Saved` instead of `Created`.
- Parent directory not created.

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
		"Remote notes/todo.md is missing; starting from an empty file",
		"Created "+resp.RemotePath,
		md5Hex("buy milk\n"),
	)
	combinedHasNone(t, resp.Combined, "Downloading", "Saved ")

	assertRemoteContent(t, resp, "notes/todo.md", "buy milk\n")
	assertRemoteMode(t, resp, "notes/todo.md", 0644)

	assert.Output(t, resp.Stdout, `---
version: 2
__EDITOR__: type=string
__STAGED__: type=string
__REMOTE__: type=string
__MD5__: type=string
---
Remote notes/todo.md is missing; starting from an empty file
Opening __EDITOR__ __STAGED__
Created __REMOTE__ (9 B, md5 __MD5__)
`)
}
```
