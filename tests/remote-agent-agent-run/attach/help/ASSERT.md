## Expected

1. Exit code 0.
2. Stdout is help for `agent-run attach` and mentions a session id placeholder
   (`session-id`, `session_id`, or `<session`).
3. Stdout is non-empty.

## Errors

- Exit non-zero.
- Help omits session id.

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
	if strings.TrimSpace(out) == "" {
		t.Fatalf("expected non-empty attach help")
	}
	lower := strings.ToLower(out)
	if !strings.Contains(lower, "attach") {
		t.Fatalf("help must mention attach; stdout:\n%s", out)
	}
	if !strings.Contains(lower, "session") {
		t.Fatalf("attach help must document session id; stdout:\n%s", out)
	}
}
```
