## Expected

1. The title suffix is `5%`.
2. The body lists cycle usage, remaining credit, requests, both windows, and renewal.

## Errors

- Reordering the parts, or using a different percent (for example the weekly window).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Title != "5%" {
		t.Fatalf("title suffix = %q, want 5%%", resp.Title)
	}
	want := "5% used, $66.19 left, 1,870 requests, 5h 15%, Weekly 11%, renews in 26d"
	if resp.Body != want {
		t.Fatalf("body = %q, want %q", resp.Body, want)
	}
}
```
