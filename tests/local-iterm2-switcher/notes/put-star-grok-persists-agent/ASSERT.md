## Expected

1. HTTP 200 ok; starred record kept.
2. Stored v2 item has grok agent fields and the live iTerm UUID.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || !resp.OK {
		t.Fatalf("status=%d ok=%v body=%s", resp.StatusCode, resp.OK, resp.Body)
	}
	if !resp.HasRecord || !resp.StoredBookmarked {
		t.Fatalf("record=%v bookmarked=%v", resp.HasRecord, resp.StoredBookmarked)
	}
	if resp.StoredItemsCount != 1 {
		t.Fatalf("items=%d want 1", resp.StoredItemsCount)
	}
	if resp.StoredAgentRunner != "grok" {
		t.Fatalf("agent_runner=%q want grok", resp.StoredAgentRunner)
	}
	if resp.StoredGrokSessionID != "g1" {
		t.Fatalf("grok_session_id=%q want g1", resp.StoredGrokSessionID)
	}
	if resp.StoredITermSessionID != "sess-a" {
		t.Fatalf("iterm_session_id=%q want sess-a", resp.StoredITermSessionID)
	}
}
```
