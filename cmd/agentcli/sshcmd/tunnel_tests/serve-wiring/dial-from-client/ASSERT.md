## Expected

1. Harness `err` is nil.
2. `WiringErr` empty.
3. `DialIsNil` is false.
4. `WiringSessID` (or `SessionID`) non-empty.
5. `TunnelDialErr` empty (smoke dial succeeded).

## Side Effects

- Remote session created for wiring; cleaned with httptest.

## Errors

None expected on happy path.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	_ = req
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.WiringErr != "" {
		t.Fatalf("BuildSSHTunnelDial: %s", resp.WiringErr)
	}
	if resp.DialIsNil {
		t.Fatal("Dial must be non-nil when Client is provided")
	}
	sid := resp.WiringSessID
	if sid == "" {
		sid = resp.SessionID
	}
	if sid == "" {
		t.Fatal("session id from BuildSSHTunnelDial must be non-empty")
	}
	if resp.TunnelDialErr != "" {
		t.Fatalf("smoke dial() failed: %s", resp.TunnelDialErr)
	}
}
```
