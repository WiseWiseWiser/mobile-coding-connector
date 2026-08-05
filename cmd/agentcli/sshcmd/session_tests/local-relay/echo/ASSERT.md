## Expected

1. `RelayStartErr` and `EchoErr` are empty.
2. `LocalPort` is greater than 0.
3. `EchoGot` equals `EchoPayload` (`"hi"`).

## Side Effects

- Ephemeral TCP listener on 127.0.0.1 closed after Run (defer Close).

## Errors

None expected.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.RelayStartErr != "" {
		t.Fatalf("LocalRelay.Start error: %s", resp.RelayStartErr)
	}
	if resp.EchoErr != "" {
		t.Fatalf("client echo error: %s", resp.EchoErr)
	}
	if resp.LocalPort <= 0 {
		t.Fatalf("LocalPort: got %d want > 0", resp.LocalPort)
	}
	want := req.EchoPayload
	if want == "" {
		want = "hi"
	}
	if resp.EchoGot != want {
		t.Fatalf("EchoGot: got %q want %q", resp.EchoGot, want)
	}
}
```
