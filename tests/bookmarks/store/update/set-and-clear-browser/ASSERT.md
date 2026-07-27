## Expected

1. After clear, node browser is nil or points to empty string (inherit).

## Errors

- Browser still firefox after clear.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.ErrMsg != "" {
		t.Fatalf("ErrMsg: %s", resp.ErrMsg)
	}
	n := FindNode(resp.Doc, "bm_b")
	if n == nil {
		t.Fatal("bm_b missing")
	}
	if n.Browser != nil && *n.Browser != "" {
		t.Fatalf("browser should be cleared, got %q", *n.Browser)
	}
}
```
