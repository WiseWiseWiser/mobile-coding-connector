## Expected

1. Exit 0 (nesting into basename; sibling `child/` does not block).
2. Files under `uploads/apps/srcdir/`.
3. `uploads/apps/child` still exists.

## Exit Code

0.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	assertServerFileContent(t, resp.ServerHome, "uploads/apps/srcdir/a.txt", "alpha\n")
	assertServerFileContent(t, resp.ServerHome, "uploads/apps/srcdir/sub/b.txt", "bravo\n")
	assertServerIsDir(t, resp.ServerHome, "uploads/apps/child")
}
```
