## Expected

1. ErrMsg non-empty.
2. No node id bm_noname.

## Errors

- Accepts empty name.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.ErrMsg == "" {
		t.Fatal("expected validation error for empty name")
	}
	if FindNode(resp.Doc, "bm_noname") != nil {
		t.Fatal("invalid node was stored")
	}
}
```
