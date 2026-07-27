## Expected

1. `TimeLeft` is exactly `left 4h5m`.

## Errors

- Omitting minutes or using `min` suffix.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.TimeLeft != "left 4h5m" {
		t.Fatalf("TimeLeft = %q, want %q", resp.TimeLeft, "left 4h5m")
	}
}
```