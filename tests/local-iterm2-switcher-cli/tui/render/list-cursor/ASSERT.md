## Expected

1. Default `FocusPane` is list.
2. View has `›` on the first session row (`grok review` → sess-a).

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.FocusPane != "list" {
		t.Fatalf("FocusPane=%q want list (default focus is list)", resp.FocusPane)
	}
	if !resp.HasListCursor {
		t.Fatalf("default › must be on first session row; view=%q", resp.ViewText)
	}
	if resp.ListCursorSession != "sess-a" {
		t.Fatalf("ListCursorSession=%q want sess-a (› on grok review)", resp.ListCursorSession)
	}
}
```
