## Expected

1. Exit 0.
2. Stdout mentions `status` and session (or home / multi-layer intent).
3. Prefer documenting `--json` when product supports it (soft if omitted).

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
	out := strings.ToLower(resp.Stdout)
	if strings.TrimSpace(out) == "" {
		t.Fatalf("expected non-empty status help")
	}
	if !strings.Contains(out, "status") {
		t.Fatalf("help must mention status; stdout:\n%s", resp.Stdout)
	}
	if !strings.Contains(out, "session") && !strings.Contains(out, "home") {
		t.Fatalf("status help should mention session id or home; stdout:\n%s", resp.Stdout)
	}
}
```
