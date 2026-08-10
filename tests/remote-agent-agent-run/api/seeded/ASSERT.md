## Expected

1. HTTP 200.
2. Three sessions in the payload with correct `session_id`, `runner`, `status`.
3. Newest-first order: `sess-new`, `sess-mid`, `sess-old`.

## Errors

- Missing fields or wrong ids.
- Non-200.

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
	if len(resp.Sessions) != 3 {
		t.Fatalf("want 3 sessions, got %d body:\n%s", len(resp.Sessions), resp.Body)
	}
	wantOrder := []string{"sess-new", "sess-mid", "sess-old"}
	wantMeta := map[string]struct{ Runner, Status string }{
		"sess-new": {"opencode", "idle"},
		"sess-mid": {"grok", "running"},
		"sess-old": {"codex", "finished"},
	}
	for i, id := range wantOrder {
		got := resp.Sessions[i]
		if got.SessionID != id {
			t.Fatalf("order[%d]=%q want %q; full=%+v", i, got.SessionID, id, resp.Sessions)
		}
		w := wantMeta[id]
		if got.Runner != w.Runner || got.Status != w.Status {
			t.Fatalf("%s runner/status=%q/%q want %q/%q", id, got.Runner, got.Status, w.Runner, w.Status)
		}
	}
}
```
