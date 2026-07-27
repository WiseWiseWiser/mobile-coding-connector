## Expected

1. `FindNode(Doc, "bm_custom")` non-nil with matching name/url.

## Errors

- Id rewritten or missing.

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
	n := FindNode(resp.Doc, "bm_custom")
	if n == nil {
		t.Fatal("bm_custom missing")
	}
	if n.Name != "Custom" || n.URL != "https://example.com/custom" {
		t.Fatalf("mismatch: %+v", n)
	}
}
```
