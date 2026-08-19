## Expected

1. TUIAction is `quit`.
2. No focus session id required (empty).

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.TUIAction != "quit" {
		t.Fatalf("TUIAction=%q want quit", resp.TUIAction)
	}
}
```
