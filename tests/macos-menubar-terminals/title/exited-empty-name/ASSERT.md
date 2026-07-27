## Expected

1. `Title` is exactly `sess-1 [EXITED]` (id base + exact exited suffix).

## Errors

- Suffix without base id, or base without suffix.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "sess-1 [EXITED]"
	if resp.Title != want {
		t.Fatalf("title = %q, want %q", resp.Title, want)
	}
}
```
