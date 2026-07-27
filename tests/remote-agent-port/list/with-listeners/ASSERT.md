## Expected

1. Exit 0.
2. Stdout includes headers or columns for PORT, PID, COMMAND (or equivalent).
3. Stdout includes seeded port `3000`, pid `4242`, command `node`.

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
	out := resp.Stdout
	for _, needle := range []string{"3000", "4242", "node"} {
		if !strings.Contains(out, needle) {
			t.Fatalf("list missing %q; stdout:\n%s", needle, out)
		}
	}
	upper := strings.ToUpper(out)
	if !strings.Contains(upper, "PORT") || !strings.Contains(upper, "PID") {
		t.Fatalf("expected PORT/PID columns; stdout:\n%s", out)
	}
}
```
