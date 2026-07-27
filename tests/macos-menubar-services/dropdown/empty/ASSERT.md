## Expected

1. `EmptyLabel` is exactly `No services configured`.

## Errors

- Missing or alternate placeholder text.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.EmptyLabel != "No services configured" {
		t.Fatalf("empty label = %q, want %q", resp.EmptyLabel, "No services configured")
	}
}
```