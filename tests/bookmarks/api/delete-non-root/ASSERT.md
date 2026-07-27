## Expected

1. 2xx.
2. bm_d absent from Doc.

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
	if FindNode(resp.Doc, "bm_d") != nil {
		t.Fatal("bm_d still present")
	}
}
```
