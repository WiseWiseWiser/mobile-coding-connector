## Expected

1. `Label` is exactly `Grok ...`.

## Errors

- Shows limit or error while still loading.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Label != "Grok ..." {
		t.Fatalf("label = %q, want %q", resp.Label, "Grok ...")
	}
}
```