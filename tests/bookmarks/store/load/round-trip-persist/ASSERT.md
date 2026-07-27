## Expected

1. After reload, `FindNode(Doc, "bm_dash")` has name Local Dashboard and matching url.
2. Root still `root`.

## Errors

- Node missing after reload; fields changed.

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
	n := FindNode(resp.Doc, "bm_dash")
	if n == nil {
		t.Fatal("bm_dash missing after reload")
	}
	if n.Name != "Local Dashboard" || n.URL != "http://127.0.0.1:7070" || n.Type != "url" {
		t.Fatalf("node mismatch: %+v", n)
	}
	if !defaultRootOK(resp.Doc) {
		t.Fatalf("root broken: %+v", resp.Doc)
	}
}
```
