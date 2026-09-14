---
explanation: "L2 --no-override conflict preflight"
---

## Expected

1. Non-zero exit.
2. Combined output mentions `--no-override` / overwritten.
3. Preseeded `uploads/apps/srcdir/a.txt` unchanged.
4. Local tree files not applied (`sub/b.txt` absent).

## Exit Code

Non-zero.

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
	if resp.ExitCode == 0 {
		t.Fatalf("expected failure; combined:\n%s", resp.Combined)
	}

	lower := strings.ToLower(resp.Combined)
	if !strings.Contains(lower, "no-override") && !strings.Contains(lower, "overwritten") {
		t.Fatalf("expected --no-override conflict error; combined:\n%s", resp.Combined)
	}

	assertServerFileContent(t, resp.ServerHome, "uploads/apps/srcdir/a.txt", "seed-existing\n")
	assertServerPathMissing(t, resp.ServerHome, "uploads/apps/srcdir/sub/b.txt")
}
```
