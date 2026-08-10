## Expected

1. No transport error.
2. HTTP 2xx.
3. Event landed in hub Recent.

## Errors

- 401 with correct token.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("status = %d, want 2xx; body=%s", resp.StatusCode, resp.Body)
	}
	if len(resp.Recent) < 1 {
		t.Fatal("hub Recent empty after authenticated publish")
	}
}
```
