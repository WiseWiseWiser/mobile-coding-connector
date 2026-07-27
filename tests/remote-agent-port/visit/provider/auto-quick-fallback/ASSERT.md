## Expected

1. StartErr empty.
2. Session.Provider is `cloudflare_quick` (or `quick`).

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.StartErr != "" {
		t.Fatalf("StartErr: %s", resp.StartErr)
	}
	if resp.Session == nil {
		t.Fatal("expected session")
	}
	p := resp.Session.Provider
	if p != "cloudflare_quick" && p != "quick" {
		t.Fatalf("provider = %q, want cloudflare_quick", p)
	}
}
```
