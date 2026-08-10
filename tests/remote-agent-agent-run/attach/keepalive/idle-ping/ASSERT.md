## Expected

1. Attach WebSocket connects (upgrade success).
2. `PingCount >= 1` within WaitForPing under zero app traffic.
3. Pings are control-plane only (test does not require PTY bytes).

## Errors

- Zero pings after wait (keepalive not wired or interval ignored).
- Failed dial.

## Exit Code

0

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.AttachErr != "" {
		t.Fatalf("keepalive attach failed: status=%d err=%q", resp.WSHTTPStatus, resp.AttachErr)
	}
	if resp.WSHTTPStatus != 0 && resp.WSHTTPStatus != 101 {
		t.Fatalf("want WS 101, got %d", resp.WSHTTPStatus)
	}
	if resp.PingCount < 1 {
		t.Fatalf("expected >=1 WebSocket Ping under idle (interval inject); PingCount=%d", resp.PingCount)
	}
}
```
