## Expected

1. `ResetDisplay` is exactly `July 9, 17:55`.

## Errors

- Leaving `PT` suffix or wrong clock time.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ResetDisplay != "July 9, 17:55" {
		t.Fatalf("ResetDisplay = %q, want %q", resp.ResetDisplay, "July 9, 17:55")
	}
}
```