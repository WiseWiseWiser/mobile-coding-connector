## Expected

1. `ServeErr` is non-empty.
2. Error text is configuration-related: contains `dial` (case-insensitive) and/or
   `not configured` / `required` / `missing`.

## Side Effects

- No Alive session should remain required for this leaf; Load may be nil (not asserted strictly).

## Errors

- Expected: Start fails without a usable Dial.

```go
import (
	"strings"
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
	if resp.ServeErr == "" {
		t.Fatal("Start without Dial must return an error; ServeErr is empty")
	}
	low := strings.ToLower(resp.ServeErr)
	hasDial := strings.Contains(low, "dial")
	hasCfg := strings.Contains(low, "not configured") ||
		strings.Contains(low, "required") ||
		strings.Contains(low, "missing") ||
		strings.Contains(low, "nil")
	if !hasDial && !hasCfg {
		t.Fatalf("ServeErr should mention dial or configuration; got %q", resp.ServeErr)
	}
}
```
