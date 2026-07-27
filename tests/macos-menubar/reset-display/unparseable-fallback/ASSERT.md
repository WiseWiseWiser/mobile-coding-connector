## Expected

1. `ResetDisplay` is exactly `soon` (unchanged raw input).

## Errors

- Empty string, error, or reformatted output.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ResetDisplay != "soon" {
		t.Fatalf("ResetDisplay = %q, want %q", resp.ResetDisplay, "soon")
	}
}
```