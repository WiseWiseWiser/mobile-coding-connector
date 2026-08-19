## Expected

1. Live session is bookmarked.
2. Note still joined.
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
	if resp.NoteOnFirst != "fix auth on staging" {
		t.Fatalf("note=%q", resp.NoteOnFirst)
	}
	if resp.HasOrphan {
		t.Fatal("unexpected orphan")
	}
}
```
