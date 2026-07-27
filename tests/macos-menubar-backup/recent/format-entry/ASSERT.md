## Expected

1. `FormattedEntry` is exactly `12m ago · 42 MB`.

## Errors

- Decimal sizes (`42.00 MB`), missing `ago`, wrong separator, or IEC `MiB` label.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "12m ago · 42 MB"
	if resp.FormattedEntry != want {
		t.Fatalf("FormattedEntry = %q, want %q", resp.FormattedEntry, want)
	}
}
```
