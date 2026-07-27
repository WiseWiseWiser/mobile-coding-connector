## Expected

1. `ResetDisplay` is exactly `July 10, 09:55`.

## Errors

- Same calendar day as Pacific source or wrong hour.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ResetDisplay != "July 10, 09:55" {
		t.Fatalf("ResetDisplay = %q, want %q", resp.ResetDisplay, "July 10, 09:55")
	}
}
```