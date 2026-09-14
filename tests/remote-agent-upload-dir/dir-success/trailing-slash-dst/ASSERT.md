## Expected

1. Exit code 0.
2. `parent/proj/file.txt` exists.
3. Staged progress + `uploaded` product.

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
	combinedHasAll(t, resp.Combined, "[1/4] resolve", "uploaded", "1 files", "parent/proj")
	assertServerPathMissing(t, resp.ServerHome, "parent/file.txt")
	assertServerFileContent(t, resp.ServerHome, "parent/proj/file.txt", "proj payload\n")
}
```
