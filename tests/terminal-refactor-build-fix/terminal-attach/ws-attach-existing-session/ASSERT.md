# Expected

When the client attaches with a known `SessionID` and the server accepts the
connection and begins serving (sends a binary output frame) without echoing a
`session_id` frame — the pre-refactor reattach behavior — `AttachWithIO` must
**proceed with the known SessionID** and succeed, not time out.

- No panic: `resp.AttachPaniced == false`.
- No error: `resp.AttachErr == ""` (in particular, not
  `timeout waiting for session_id message`).
- The returned `AttachResult.SessionID` is non-empty (the client proceeds with
  the SessionID it supplied: `resp.AttachSessionID == resp.AttachKnownSessionID`).

The fixed behavior: `readSessionID` (or its caller) treats a server that
begins serving without a `session_id` echo as an accepted attach when the
caller already supplied a `SessionID`, instead of demanding the echo and
timing out. This restores backward compatibility with pre-refactor daemons
that omit `session_id` on reattach.

Contrast with `ws-attach-no-session-id`: there the caller supplies no
`SessionID`, so a silent server correctly yields a timeout. Here the caller
supplies a `SessionID`, so the client should not require the echo.

```go
import (
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if resp.AttachPaniced {
		t.Fatalf("attach must not panic, but panicked: %s", resp.AttachPanic)
	}
	if resp.AttachErr != "" {
		t.Fatalf("attach with a known SessionID must succeed against a serving server, got error: %q", resp.AttachErr)
	}
	if resp.AttachSessionID == "" {
		t.Fatal("expected attach to proceed with the known SessionID, got empty ID (timeout)")
	}
	if resp.AttachSessionID != resp.AttachKnownSessionID {
		t.Fatalf("expected session ID %q, got %q", resp.AttachKnownSessionID, resp.AttachSessionID)
	}
}
```
