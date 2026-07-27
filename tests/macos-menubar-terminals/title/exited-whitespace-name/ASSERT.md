## Expected

1. `Title` is exactly `sess-1 [EXITED]` (whitespace name discarded; exited suffix applied).

## Errors

- Treating whitespace as a visible name, or omitting the exited suffix.

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
