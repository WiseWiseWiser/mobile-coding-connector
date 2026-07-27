## Expected

1. `Label` is exactly `Open in Browser`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Label != "Open in Browser" {
		t.Fatalf("label = %q, want %q", resp.Label, "Open in Browser")
	}
}
```