## Expected

1. Attach fails (non-zero ExitCode and/or non-empty AttachErr).
2. HTTP upgrade status is 4xx (typically 404) **or** error text mentions not
   found / unknown / unavailable / missing.
3. Completes quickly (no hang) — Run returns.

## Errors

- Successful WS attach (101 + empty error).

## Exit Code

non-zero (attach failure)

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
		t.Fatalf("missing session must not attach successfully; resp=%+v", resp)
	}
	if resp.WSHTTPStatus == 101 {
		t.Fatalf("missing session must not upgrade WS; status=101 err=%q", resp.AttachErr)
	}
	// Prefer 404/400, but accept any non-success with a clear message.
	msg := strings.ToLower(resp.AttachErr + " " + resp.Body + " " + resp.Combined)
	if resp.WSHTTPStatus != 0 && resp.WSHTTPStatus != 404 && resp.WSHTTPStatus != 400 &&
		resp.WSHTTPStatus != 401 && resp.WSHTTPStatus < 400 {
		// still require textual signal
		if !strings.Contains(msg, "not found") && !strings.Contains(msg, "unknown") &&
			!strings.Contains(msg, "unavailable") && !strings.Contains(msg, "missing") &&
			!strings.Contains(msg, "no such") && !strings.Contains(msg, "404") {
			t.Fatalf("expected clear missing-session error; status=%d msg=%q", resp.WSHTTPStatus, msg)
		}
	}
	if resp.WSHTTPStatus == 0 && msg == "" {
		t.Fatalf("expected attach error details; resp=%+v", resp)
	}
}
```
