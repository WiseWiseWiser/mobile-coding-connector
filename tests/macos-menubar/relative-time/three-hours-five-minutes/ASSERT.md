## Expected

1. `TimeLeft` is exactly `left 3h5m`.

## Errors

- Single-unit `left 3h` only, or wrong minute count.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.TimeLeft != "left 3h5m" {
		t.Fatalf("TimeLeft = %q, want %q", resp.TimeLeft, "left 3h5m")
	}
}
```