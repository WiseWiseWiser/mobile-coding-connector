## Expected Output

```
Downloading app.conf -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
warning: __REMOTE__ is a symlink; wrote __RESOLVED__
Saved __REMOTE__ (4 B, md5 __MD5__)
```

## Expected

1. Exit code 0.
2. Stderr carries `warning: <requested> is a symlink; wrote <target>` (the format
   `Saved <requested>` still names the path the user asked for).
3. `app.conf` is still a symlink and `real/app.conf` contains `new\n`.
4. The symlink is not replaced by a regular file.

## Side Effects

- Symlink target replaced; the symlink itself untouched.

## Errors

- Link replaced by a regular file (target resolved with `os.Rename` over the link).
- No warning, leaving the user unaware which file was written.

## Exit Code

0.

```go
import (
	"path/filepath"
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

	resolved := filepath.Join(resp.ServerHome, "real", "app.conf")
	combinedHasAll(t, resp.Combined,
		"warning: "+resp.RemotePath+" is a symlink; wrote "+resolved,
		"Saved "+resp.RemotePath,
	)
	if !strings.Contains(resp.Stderr, "warning: ") {
		t.Fatalf("warning should go to stderr;\nstderr:\n%s", resp.Stderr)
	}

	assertRemoteIsSymlink(t, resp, "app.conf")
	assertRemoteContent(t, resp, "real/app.conf", "new\n")

	assert.Output(t, resp.Stdout, `---
version: 2
__STAGED__: type=string
__EDITOR__: type=string
__REMOTE__: type=string
__MD5__: type=string
---
Downloading app.conf -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (4 B, md5 __MD5__)
`)
}
```
