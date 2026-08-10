## Expected

1. First server started (`FirstStarted` true).
2. `SecondErr` is non-nil (bind failure).

## Errors

- Second start succeeds (port reuse silently).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	// Run itself should not fail on first start; second error is in Response.
	if err != nil {
		t.Fatalf("Run error (first start path): %v", err)
	}
	if !resp.FirstStarted {
		t.Fatal("first StartPublishServer did not start")
	}
	if resp.SecondErr == nil {
		t.Fatalf("second StartPublishServer on %q succeeded; want error", resp.ListenAddr)
	}
}
```
