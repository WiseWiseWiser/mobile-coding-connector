## Expected

1. Exit code 0.
2. Files land under `uploads/apps/srcdir/`.
3. Stdout product contains `uploaded` and `2 files`.

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

	assertStdoutEndsWithNewline(t, resp.Stdout)
	combinedHasAll(t, resp.Combined, "[1/4] resolve", "uploaded", "2 files", "srcdir")
	assertServerFileContent(t, resp.ServerHome, "uploads/apps/srcdir/a.txt", "alpha\n")
	assertServerFileContent(t, resp.ServerHome, "uploads/apps/srcdir/sub/b.txt", "bravo\n")
}
```
