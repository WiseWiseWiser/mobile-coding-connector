## Expected

1. `TimeLeft` is exactly `left 1m`.

## Errors

- Returning `left 2m` (rounding up) or `left 0m`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.TimeLeft != "left 1m" {
		t.Fatalf("TimeLeft = %q, want %q", resp.TimeLeft, "left 1m")
	}
}
```