## Expected

1. Node named Dev, type folder, children length 0.
2. Under root.

## Errors

- Treated as url; non-empty children unexpectedly.

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
	n := FindNodeByName(resp.Doc, "Dev")
	if n == nil {
		t.Fatal("Dev folder missing")
	}
	if n.Type != "folder" {
		t.Fatalf("type=%s want folder", n.Type)
	}
	if len(n.Children) != 0 {
		t.Fatalf("want empty children, got %d", len(n.Children))
	}
}
```
