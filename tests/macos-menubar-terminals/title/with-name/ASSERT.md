## Expected

1. `Title` is exactly `demo` (name wins over id).

## Errors

- Using id when name is non-empty, or appending cwd/status to title.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Title != "demo" {
		t.Fatalf("title = %q, want %q", resp.Title, "demo")
	}
}
```
