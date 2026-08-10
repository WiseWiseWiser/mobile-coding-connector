## Expected

1. No transport error (HTTP response received).
2. Status code 401.

## Errors

- 2xx without auth when token configured.

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
	if resp.StatusCode != 401 {
		t.Fatalf("status = %d, want 401; body=%s", resp.StatusCode, resp.Body)
	}
}
```
