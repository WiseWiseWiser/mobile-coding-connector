## Expected

1. `Label` is exactly `Open in Browser(Firefox)`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Open in Browser(Firefox)"
	if resp.Label != want {
		t.Fatalf("label = %q, want %q", resp.Label, want)
	}
}
```