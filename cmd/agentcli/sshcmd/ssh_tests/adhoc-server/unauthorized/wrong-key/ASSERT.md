## Expected

1. Harness `err` is nil.
2. `AuthErr` is non-empty (public-key auth rejected).
3. `AdhocPort` > 0 (server did start).

## Side Effects

- AdhocServer closed after Run.

## Errors

- Expected: SSH authentication / handshake failure on client dial.

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
	if resp.AdhocPort <= 0 {
		t.Fatalf("AdhocPort: got %d want > 0", resp.AdhocPort)
	}
	if resp.AuthErr == "" {
		t.Fatal("expected AuthErr when dialing with unauthorized key; got empty")
	}
}
```
