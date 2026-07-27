## Expected

1. 2xx status.
2. Node bm_p name Renamed; url unchanged.

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
	n := FindNode(resp.Doc, "bm_p")
	if n == nil || n.Name != "Renamed" {
		t.Fatalf("rename failed: %+v", n)
	}
	if n.URL != "https://p.example.com" {
		t.Fatalf("url changed unexpectedly: %s", n.URL)
	}
}
```
