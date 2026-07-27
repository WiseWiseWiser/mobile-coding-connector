## Expected

1. `Title` is exactly `api ○ Stopped (disabled)`.

## Errors

- Missing `(disabled)` suffix or wrong hollow indicator.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "api ○ Stopped (disabled)"
	if resp.Title != want {
		t.Fatalf("title = %q, want %q", resp.Title, want)
	}
}
```