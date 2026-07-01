# Expected

When the `/api/terminal` server upgrades the WebSocket but never sends a
`session_id` message, `AttachWithIO` must surface a **graceful timeout error**
and must **not panic**.

- No panic: `resp.AttachPaniced == false`.
- A non-nil error is returned whose text mentions `timeout` or `session`.
- No session ID is reported: `resp.AttachSessionID == ""`.

The fixed behavior: `readSessionID` must treat a read-deadline timeout on a
poisoned (permanently failed) gorilla connection as terminal, returning the
`"timeout waiting for session_id message"` error — not `continue`-ing into a
tight spin that trips gorilla's `readErrCount >= 1000` panic guard.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if resp.AttachPaniced {
		t.Fatalf("attach must not panic, but panicked: %s", resp.AttachPanic)
	}
	if resp.AttachErr == "" {
		t.Fatal("expected a graceful timeout error from attach, got nil error")
	}
	low := strings.ToLower(resp.AttachErr)
	if !strings.Contains(low, "timeout") && !strings.Contains(low, "session") {
		t.Fatalf("error %q should indicate timeout/session_id", resp.AttachErr)
	}
	if resp.AttachSessionID != "" {
		t.Fatalf("expected empty session ID, got %q", resp.AttachSessionID)
	}
}
```
