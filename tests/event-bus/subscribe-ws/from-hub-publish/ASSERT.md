## Expected

1. No error.
2. Exactly one WS event received.
3. Type/source match request; id non-empty.

## Errors

- No frame / wrong payload.

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
