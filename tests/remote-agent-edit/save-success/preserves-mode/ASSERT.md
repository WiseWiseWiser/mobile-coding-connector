## Expected Output

```
Downloading run.sh -> __STAGED__ (10 B)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (18 B, md5 __MD5__)
```

## Expected

1. Exit code 0 and a `Saved` line.
2. Remote `run.sh` contains the new script text.
3. Remote mode is still `0755` (preserved from the download-time file).

## Side Effects

- File replaced atomically; mode carried over.

## Errors

- Mode reset to 0644 by the write path.

## Exit Code

0.

```go
import (
	"os"
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
	assertRemoteContent(t, resp, "run.sh", "#!/bin/sh\necho hi\n")
	assertRemoteMode(t, resp, "run.sh", os.FileMode(0755))

	assert.Output(t, resp.Stdout, `---
version: 2
__STAGED__: type=string
__EDITOR__: type=string
__REMOTE__: type=string
__MD5__: type=string
---
Downloading run.sh -> __STAGED__ (10 B)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (18 B, md5 __MD5__)
`)
}
```
