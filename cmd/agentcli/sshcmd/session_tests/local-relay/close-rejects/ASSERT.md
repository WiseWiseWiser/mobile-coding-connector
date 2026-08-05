## Expected

1. `RelayStartErr` is empty.
2. `LocalPort` was greater than 0 before close.
3. `DialAfterCloseErr` is non-empty (connection refused / timeout / similar).
4. `RelayCloseErr` is empty (clean Close).

## Side Effects

- Listener no longer accepts on the recorded port.

## Errors

- Expected: dial after Close fails.

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
	if resp.RelayStartErr != "" {
		t.Fatalf("LocalRelay.Start error: %s", resp.RelayStartErr)
	}
	if resp.LocalPort <= 0 {
		t.Fatalf("LocalPort before close: got %d want > 0", resp.LocalPort)
	}
	if resp.RelayCloseErr != "" {
		t.Fatalf("LocalRelay.Close error: %s", resp.RelayCloseErr)
	}
	if resp.DialAfterCloseErr == "" {
		t.Fatal("dial after Close must fail; DialAfterCloseErr is empty")
	}
}
```
