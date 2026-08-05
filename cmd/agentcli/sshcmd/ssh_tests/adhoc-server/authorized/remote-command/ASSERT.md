## Expected

1. Harness `err` is nil.
2. `AuthErr` and `CommandErr` are empty.
3. `AdhocPort` > 0.
4. `Stdout` contains `EchoNeedle` (`hello`).

## Side Effects

- Ephemeral AdhocServer listener closed after Run (t.Cleanup).

## Errors

None expected.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.AuthErr != "" {
		t.Fatalf("SSH auth error: %s", resp.AuthErr)
	}
	if resp.CommandErr != "" {
		t.Fatalf("remote command error: %s", resp.CommandErr)
	}
	if resp.AdhocPort <= 0 {
		t.Fatalf("AdhocPort: got %d want > 0", resp.AdhocPort)
	}
	needle := req.EchoNeedle
	if needle == "" {
		needle = "hello"
	}
	if !strings.Contains(resp.Stdout, needle) {
		t.Fatalf("Stdout must contain %q; got %q (stderr=%q)", needle, resp.Stdout, resp.Stderr)
	}
}
```
