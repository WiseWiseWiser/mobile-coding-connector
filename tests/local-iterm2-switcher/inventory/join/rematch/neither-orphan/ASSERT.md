## Expected

1. Live session is not bookmarked.
2. One saved_notes orphan with the unmatched item note.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.FirstBookmarked || resp.BookmarkCount != 0 {
		t.Fatalf("bookmarked=%v count=%d", resp.FirstBookmarked, resp.BookmarkCount)
	}
	if resp.NoteOnFirst != "" {
		t.Fatalf("live note=%q want empty", resp.NoteOnFirst)
	}
	if !resp.HasOrphan || resp.SavedCount != 1 {
		t.Fatalf("SavedCount=%d HasOrphan=%v", resp.SavedCount, resp.HasOrphan)
	}
	if resp.OrphanNote != "staging fix" {
		t.Fatalf("orphan note=%q", resp.OrphanNote)
	}
}
```
