## Expected

1. No error.
2. `Published.ID` equals request ID.
3. `Published.TS` equals request TS.

## Errors

- Hub overwrites non-empty id/ts.

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
	if resp.Published.ID != req.Event.ID {
		t.Fatalf("ID = %q, want %q", resp.Published.ID, req.Event.ID)
	}
	if resp.Published.TS != req.Event.TS {
		t.Fatalf("TS = %q, want %q", resp.Published.TS, req.Event.TS)
	}
}
```
