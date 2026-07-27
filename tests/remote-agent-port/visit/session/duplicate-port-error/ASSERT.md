## Expected

1. First Start succeeds (Session non-nil).
2. Second Start sets StartErr non-empty.
3. Only one active session remains.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.Session == nil {
		t.Fatal("first Start should succeed")
	}
	if resp.StartErr == "" {
		t.Fatal("second Start on same port must error")
	}
	if len(resp.Sessions) != 1 {
		t.Fatalf("want 1 active session, got %d", len(resp.Sessions))
	}
}
```
