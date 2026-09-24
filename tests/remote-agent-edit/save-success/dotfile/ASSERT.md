## Expected Output

```
Downloading .env -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (4 B, md5 __MD5__)
```

## Expected

1. Exit code 0 and a `Saved` line naming the `.env` path.
2. The staged copy is `<staging>/<serverHome>/.env`.
3. Remote `.env` contains `A=2\n`.

## Side Effects

- Remote `.env` replaced.

## Errors

- Dotfile skipped or staged under a different name.

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
		"Downloading .env -> "+resp.StagedPath+" (4 B)",
		"Saved "+resp.RemotePath,
	)
	assertStagedPathUnderStaging(t, resp)
	assertRemoteContent(t, resp, ".env", "A=2\n")

	assert.Output(t, resp.Stdout, `---
version: 2
__STAGED__: type=string
__EDITOR__: type=string
__REMOTE__: type=string
__MD5__: type=string
---
Downloading .env -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (4 B, md5 __MD5__)
`)
}
```
