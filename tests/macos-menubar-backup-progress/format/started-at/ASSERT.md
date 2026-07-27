## Expected

1. `Line` is exactly `Started 2026-07-10 15:00:00`.

## Errors

- RFC3339 with `T`/`Z`, wrong order, or relative times.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Started 2026-07-10 15:00:00"
	if resp.Line != want {
		t.Fatalf("Line = %q, want %q", resp.Line, want)
	}
}
```
