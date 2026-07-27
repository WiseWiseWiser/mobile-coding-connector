## Expected

1. StartErr non-empty (no available provider).
2. Session is nil.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.StartErr == "" {
		t.Fatal("expected StartErr when no provider available")
	}
	if resp.Session != nil {
		t.Fatalf("session should be nil on error, got %+v", resp.Session)
	}
}
```
