## Expected

1. Live row takes the agent-matched item (not the UUID-only item).
2. The unused UUID item is a saved_notes orphan.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.FirstBookmarked || resp.BookmarkCount != 1 {
		t.Fatalf("bookmarked=%v count=%d", resp.FirstBookmarked, resp.BookmarkCount)
	}
	if resp.NoteOnFirst != "agent-item" {
		t.Fatalf("note=%q want agent-item", resp.NoteOnFirst)
	}
	if !resp.HasOrphan || resp.SavedCount != 1 {
		t.Fatalf("SavedCount=%d HasOrphan=%v", resp.SavedCount, resp.HasOrphan)
	}
	if resp.OrphanNote != "uuid-item" {
		t.Fatalf("orphan note=%q want uuid-item", resp.OrphanNote)
	}
}
```
