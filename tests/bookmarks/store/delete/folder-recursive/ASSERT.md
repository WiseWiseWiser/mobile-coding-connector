## Expected

1. fld_dev and bm_inner both absent.
2. Root remains.

## Errors

- Orphan child remains.

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
	if FindNode(resp.Doc, "fld_dev") != nil {
		t.Fatal("fld_dev still present")
	}
	if FindNode(resp.Doc, "bm_inner") != nil {
		t.Fatal("bm_inner still present after recursive delete")
	}
	if !defaultRootOK(resp.Doc) {
		t.Fatal("root missing")
	}
}
```
