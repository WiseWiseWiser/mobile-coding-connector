## Expected

1. `Title` is exactly `sess-1` (id used when name empty).

## Errors

- Empty title, placeholder text, or short-id truncation not specified by contract.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Title != "sess-1" {
		t.Fatalf("title = %q, want %q", resp.Title, "sess-1")
	}
}
```
