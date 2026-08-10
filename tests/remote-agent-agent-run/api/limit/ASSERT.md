## Expected

1. HTTP 200.
2. Exactly one session in `sessions`.
3. That session is the newest: `sess-new`.

## Errors

- More than one session returned.
- Wrong id.

## Exit Code

0.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.HTTPStatus != 200 {
		t.Fatalf("HTTP %d body:\n%s", resp.HTTPStatus, resp.Body)
	}
	if len(resp.Sessions) != 1 {
		t.Fatalf("want 1 session for ?limit=1, got %d: %+v", len(resp.Sessions), resp.Sessions)
	}
	if resp.Sessions[0].SessionID != "sess-new" {
		t.Fatalf("want sess-new, got %+v", resp.Sessions[0])
	}
}
```
