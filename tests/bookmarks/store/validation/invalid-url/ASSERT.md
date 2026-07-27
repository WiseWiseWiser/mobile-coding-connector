## Expected

1. ErrMsg non-empty.
2. bm_badurl not stored.

## Errors

- Accepts relative URL.

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
		t.Fatal("expected validation error for invalid url")
	}
	if FindNode(resp.Doc, "bm_badurl") != nil {
		t.Fatal("invalid url node was stored")
	}
}
```
