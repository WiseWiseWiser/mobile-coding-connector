## Expected

1. Non-zero exit.
2. Combined/stderr indicates missing session id or usage (mentions session /
   attach / argument / usage).

## Errors

- Exit 0.
- Silent failure.

## Exit Code

non-zero

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
		t.Fatalf("expected non-zero exit without session id; stdout:\n%s", resp.Stdout)
	}
	combined := strings.ToLower(resp.Combined + "\n" + resp.Stderr)
	if !strings.Contains(combined, "session") && !strings.Contains(combined, "usage") &&
		!strings.Contains(combined, "argument") && !strings.Contains(combined, "require") {
		t.Fatalf("expected clear missing-session-id message; combined:\n%s", resp.Combined)
	}
}
```
