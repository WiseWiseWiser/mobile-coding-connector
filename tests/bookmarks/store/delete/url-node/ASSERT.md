## Expected

1. bm_gone absent; bm_stay present; ErrMsg empty.

## Errors

- Sibling deleted; target still present.

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
	if FindNode(resp.Doc, "bm_gone") != nil {
		t.Fatal("bm_gone still present")
	}
	if FindNode(resp.Doc, "bm_stay") == nil {
		t.Fatal("bm_stay missing")
	}
}
```
