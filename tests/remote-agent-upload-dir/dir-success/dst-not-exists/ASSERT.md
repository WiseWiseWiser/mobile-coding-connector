## Expected

1. Exit code 0.
2. Staged stderr spine `[1/4]…[4/4]` with kinds resolve/pack/upload/apply.
3. Stdout product line `uploaded …` with `2 files`.
4. `uploads/mirror/a.txt` and `uploads/mirror/sub/b.txt` exist.

## Exit Code

0.

```go
import (
	"regexp"
	"testing"

	"github.com/xhd2015/doctest/session"
)

var reStageMarker = regexp.MustCompile(`(?m)^\[[1-4]/4\] `)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	assertStdoutEndsWithNewline(t, resp.Stdout)
	combinedHasAll(t, resp.Combined, "[1/4] resolve", "[2/4] pack", "[3/4] upload", "[4/4] apply", "uploaded", "2 files", "uploads/mirror")
	if n := len(reStageMarker.FindAllStringIndex(resp.Stderr, -1)); n != 4 {
		t.Fatalf("want 4 stage markers on stderr, got %d;\n%s", n, resp.Stderr)
	}
	assertServerFileContent(t, resp.ServerHome, "uploads/mirror/a.txt", "alpha\n")
	assertServerFileContent(t, resp.ServerHome, "uploads/mirror/sub/b.txt", "bravo\n")
}
```
