## Expected

1. StartErr non-empty.
2. Does not silently fall back to quick.

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
		t.Fatal("expected error for explicit owned when unavailable")
	}
	if resp.Session != nil && (resp.Session.Provider == "cloudflare_quick" || resp.Session.Provider == "quick") {
		t.Fatal("must not fall back to quick for explicit owned")
	}
}
```
