## Expected

1. `Title` is exactly `web ● Running`.

## Errors

- Wrong indicator, spacing, or status label.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Title != "web ● Running" {
		t.Fatalf("title = %q, want %q", resp.Title, "web ● Running")
	}
}
```