## Expected

1. Harness `err` is nil.
2. `CreateErr` empty; `SessionID` was non-empty before destroy.
3. `TunnelDialErr` is non-empty after destroy.
4. `DestroyErr` may be empty (successful destroy) or non-empty only if destroy itself fails hard — prefer empty DestroyErr and failed dial.

## Side Effects

- Session removed from Manager; further tunnels rejected.

## Errors

SSHTunnelDial after destroy must not return a usable conn.

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
	if resp.CreateErr != "" {
		t.Fatalf("CreateSSHSession: %s", resp.CreateErr)
	}
	if resp.SessionID == "" {
		t.Fatal("SessionID required before destroy")
	}
	if resp.TunnelDialErr == "" {
		t.Fatal("SSHTunnelDial after Destroy must fail (TunnelDialErr empty)")
	}
}
```
