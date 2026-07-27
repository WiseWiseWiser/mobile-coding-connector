## Expected

1. 2xx.
2. bm_mv under fld_api children.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.HTTPStatus < 200 || resp.HTTPStatus >= 300 {
		t.Fatalf("status=%d body=%s", resp.HTTPStatus, resp.Body)
	}
	fld := FindNode(resp.Doc, "fld_api")
	if fld == nil {
		t.Fatal("fld_api missing")
	}
	for _, c := range fld.Children {
		if c.ID == "bm_mv" {
			return
		}
	}
	t.Fatal("bm_mv not under fld_api")
}
```
