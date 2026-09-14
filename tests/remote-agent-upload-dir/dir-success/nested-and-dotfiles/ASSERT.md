## Expected

1. Exit code 0.
2. Staged spine + `uploaded` with `2 files`.
3. Dotfiles and empty `emptydir/` exist remotely.

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
	combinedHasAll(t, resp.Combined, "[1/4] resolve", "uploaded", "2 files", "uploads/dot-mirror")

	assertServerFileContent(t, resp.ServerHome, "uploads/dot-mirror/.hidden", "dotfile\n")
	assertServerFileContent(t, resp.ServerHome, "uploads/dot-mirror/sub/.keep", "")
	assertServerIsDir(t, resp.ServerHome, "uploads/dot-mirror/emptydir")
	assertServerDirEmpty(t, resp.ServerHome, "uploads/dot-mirror/emptydir")
}
```
