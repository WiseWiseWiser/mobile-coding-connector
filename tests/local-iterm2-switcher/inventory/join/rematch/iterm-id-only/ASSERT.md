## Expected

1. Live `sess-a` is bookmarked via iterm_session_id only.
2. Note is joined onto the live row.
3. No orphan.

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
	if resp.NoteOnFirst != "staging fix" {
		t.Fatalf("note=%q", resp.NoteOnFirst)
	}
	if resp.HasOrphan {
		t.Fatal("unexpected orphan")
	}
}
```
