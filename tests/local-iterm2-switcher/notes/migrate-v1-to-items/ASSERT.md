## Expected

1. Loaded document is version 2 with one item.
2. Item has `iterm_session_id` only — no agent fields.
3. Join still bookmarks the live UUID.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.DocVersion != 2 {
		t.Fatalf("version=%d want 2", resp.DocVersion)
	}
	if resp.StoredItemsCount != 1 {
		t.Fatalf("items=%d want 1", resp.StoredItemsCount)
	}
	if resp.StoredITermSessionID != "sess-a" {
		t.Fatalf("iterm_session_id=%q", resp.StoredITermSessionID)
	}
	if resp.StoredAgentRunner != "" || resp.StoredGrokSessionID != "" {
		t.Fatalf("agent_runner=%q grok_session_id=%q want empty", resp.StoredAgentRunner, resp.StoredGrokSessionID)
	}
	if !resp.HasRecord || resp.StoredNote != "staging fix" || !resp.StoredBookmarked {
		t.Fatalf("record=%v note=%q bookmarked=%v", resp.HasRecord, resp.StoredNote, resp.StoredBookmarked)
	}
	if !resp.FirstBookmarked || resp.HasOrphan {
		t.Fatalf("bookmarked=%v orphan=%v", resp.FirstBookmarked, resp.HasOrphan)
	}
}
```
