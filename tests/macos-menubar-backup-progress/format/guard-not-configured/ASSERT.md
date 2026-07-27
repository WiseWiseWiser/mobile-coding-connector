## Expected

1. `Line` is exactly `ERROR: not configured`.

## Errors

- Silent no-op, different wording (`Not configured` without ERROR), or alert-only.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "ERROR: not configured"
	if resp.Line != want {
		t.Fatalf("Line = %q, want %q", resp.Line, want)
	}
}
```
