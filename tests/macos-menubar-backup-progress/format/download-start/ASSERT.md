## Expected

1. `Line` is exactly `Downloading archive…` (Unicode ellipsis).

## Errors

- Three ASCII dots `...`, different casing, or missing word archive.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Downloading archive…"
	if resp.Line != want {
		t.Fatalf("Line = %q, want %q", resp.Line, want)
	}
}
```
