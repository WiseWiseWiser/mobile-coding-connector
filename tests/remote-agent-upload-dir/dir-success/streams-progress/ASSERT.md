---
explanation: "L2 staged directory upload streams chunk body under upload stage"
---

## Expected

1. Exit code 0.
2. Exactly four `[n/4]` markers on stderr.
3. Chunk progress appears before apply marker.
4. Remote files exist under `uploads/stream-mirror/`.

## Exit Code

0.

```go
import (
	"strings"
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
	combinedHasAll(t, resp.Combined, "[1/4] resolve", "[2/4] pack", "[3/4] upload", "[4/4] apply", "chunk", "uploaded", "uploads/stream-mirror")

	uploadIdx := strings.Index(resp.Stderr, "[3/4] upload")
	applyIdx := strings.Index(resp.Stderr, "[4/4] apply")
	if uploadIdx < 0 || applyIdx < 0 || applyIdx <= uploadIdx {
		t.Fatalf("upload stage must precede apply;\n%s", resp.Stderr)
	}
	mid := resp.Stderr[uploadIdx:applyIdx]
	if !strings.Contains(mid, "chunk") {
		t.Fatalf("chunk progress must appear under upload stage;\n%s", resp.Stderr)
	}

	assertServerFileContent(t, resp.ServerHome, "uploads/stream-mirror/a.txt", "alpha\n")
	assertServerFileContent(t, resp.ServerHome, "uploads/stream-mirror/sub/b.txt", "bravo\n")
}
```
