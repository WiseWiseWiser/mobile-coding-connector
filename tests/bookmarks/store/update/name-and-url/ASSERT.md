## Expected

1. Node bm_u name New Name, url https://new.example.com.

## Errors

- Fields unchanged or ErrMsg set.

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
	n := FindNode(resp.Doc, "bm_u")
	if n == nil {
		t.Fatal("bm_u missing")
	}
	if n.Name != "New Name" || n.URL != "https://new.example.com" {
		t.Fatalf("got %+v", n)
	}
}
```
