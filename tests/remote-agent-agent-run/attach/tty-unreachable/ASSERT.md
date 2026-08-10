## Expected

1. Attach fails (non-zero ExitCode and/or AttachErr).
2. Must not complete as successful live attach (no 101 success with empty error).
3. Error / body indicates unavailable / unreachable / not found / tty (clear).

## Errors

- Silent success.
- Hang (Run never returns).

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
	if resp.ExitCode == 0 && resp.AttachErr == "" && resp.WSHTTPStatus == 101 {
		t.Fatalf("unreachable TTY must not attach successfully")
	}
	if resp.WSHTTPStatus == 101 && resp.AttachErr == "" {
		t.Fatalf("unreachable TTY must not keep a successful WS; status=101")
	}
	msg := strings.ToLower(resp.AttachErr + " " + resp.Body + " " + resp.Combined)
	if msg == "" && resp.WSHTTPStatus == 0 {
		t.Fatalf("expected error details for unreachable TTY; resp=%+v", resp)
	}
	// Soft keyword check — product phrasing may vary.
	if msg != "" && !strings.Contains(msg, "unavail") && !strings.Contains(msg, "unreachable") &&
		!strings.Contains(msg, "not found") && !strings.Contains(msg, "tty") &&
		!strings.Contains(msg, "terminal") && !strings.Contains(msg, "dead") &&
		!strings.Contains(msg, "connect") && resp.WSHTTPStatus < 400 {
		t.Fatalf("expected clear unreachable-TTY signal; status=%d msg=%q", resp.WSHTTPStatus, msg)
	}
}
```
