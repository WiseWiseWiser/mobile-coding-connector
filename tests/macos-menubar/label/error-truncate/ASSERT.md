## Expected

1. `Label` is exactly `Grok err` (short fixed label for any error length).

## Errors

- Long error truncated in menu bar instead of fixed `Grok err`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Grok err"
	if resp.Label != want {
		t.Fatalf("label = %q, want %q", resp.Label, want)
	}
}
```