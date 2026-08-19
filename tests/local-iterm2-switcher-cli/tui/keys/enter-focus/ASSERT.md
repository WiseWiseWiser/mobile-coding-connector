## Expected

1. TUIAction is `focus`.
2. TUIActionSession is `sess-a` (default selection).

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.TUIAction != "focus" {
		t.Fatalf("TUIAction=%q want focus", resp.TUIAction)
	}
	if resp.TUIActionSession != "sess-a" {
		t.Fatalf("TUIActionSession=%q want sess-a", resp.TUIActionSession)
	}
}
```
