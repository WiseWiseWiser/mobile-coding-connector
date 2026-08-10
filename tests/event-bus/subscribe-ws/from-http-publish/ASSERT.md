## Expected

1. No error.
2. One WS event with matching type/source and non-empty id.

## Errors

- HTTP publish path not wired into hub fan-out.

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
	if len(resp.WSEvents) != 1 {
		t.Fatalf("WSEvents len = %d, want 1", len(resp.WSEvents))
	}
	ev := resp.WSEvents[0]
	if ev.ID == "" {
		t.Fatal("WS event id empty")
	}
	if ev.Type != req.Event.Type {
		t.Fatalf("type = %q, want %q", ev.Type, req.Event.Type)
	}
	if ev.Source != req.Event.Source {
		t.Fatalf("source = %q, want %q", ev.Source, req.Event.Source)
	}
}
```
