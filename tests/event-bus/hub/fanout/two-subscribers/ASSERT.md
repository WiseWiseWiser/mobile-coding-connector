## Expected

1. No error.
2. Exactly two subscriber receive lists, each length 1.
3. Both received events match Published id/type/source.

## Errors

- Only one subscriber got the event.
- Timeout / Run error.

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
	if len(resp.Received) != 2 {
		t.Fatalf("subscribers = %d, want 2", len(resp.Received))
	}
	for i, list := range resp.Received {
		if len(list) != 1 {
			t.Fatalf("subscriber %d got %d events, want 1", i, len(list))
		}
		ev := list[0]
		if ev.ID == "" {
			t.Fatalf("subscriber %d empty id", i)
		}
		if ev.Type != req.Event.Type {
			t.Fatalf("subscriber %d type = %q, want %q", i, ev.Type, req.Event.Type)
		}
		if resp.Published.ID != "" && ev.ID != resp.Published.ID {
			t.Fatalf("subscriber %d id = %q, want published %q", i, ev.ID, resp.Published.ID)
		}
	}
}
```
