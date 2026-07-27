## Expected

1. fld_m.Children contains bm_m (id).
2. Root children do not include bm_m as direct child.

## Errors

- Still under root; wrong parent.

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
	fld := FindNode(resp.Doc, "fld_m")
	if fld == nil {
		t.Fatal("fld_m missing")
	}
	found := false
	for _, c := range fld.Children {
		if c.ID == "bm_m" {
			found = true
		}
	}
	if !found {
		t.Fatal("bm_m not under fld_m")
	}
	for _, c := range RootChildren(resp.Doc) {
		if c.ID == "bm_m" {
			t.Fatal("bm_m still direct child of root")
		}
	}
}
```
