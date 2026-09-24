## Expected Output

```
Downloading ~/notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (5 B, md5 __MD5__)
```

## Expected

1. Exit code 0; the `Downloading` line echoes the user's `~/notes.md` argument.
2. The staged copy is `<staging>/<absolute resolved remote path>`.
3. `Saved` reports the resolved absolute remote path, not `~/notes.md`.
4. Remote `notes.md` contains `home\n`.

## Side Effects

- Remote `notes.md` replaced.

## Errors

- `~/` kept literally (staging under a literal `~` directory, or a 404 for the
  remote path).

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
		"Downloading ~/notes.md -> "+resp.StagedPath+" (4 B)",
		"Saved "+resp.RemotePath,
	)
	assertStagedPathUnderStaging(t, resp)
	if !strings.HasSuffix(resp.RemotePath, "/notes.md") {
		t.Fatalf("remote path = %q, want the server-home path", resp.RemotePath)
	}
	assertRemoteContent(t, resp, "notes.md", "home\n")

	assert.Output(t, resp.Stdout, `---
version: 2
__STAGED__: type=string
__EDITOR__: type=string
__REMOTE__: type=string
__MD5__: type=string
---
Downloading ~/notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (5 B, md5 __MD5__)
`)
}
```
