## Expected

1. Second GET is from cache (no extra capture).
2. Live session is bookmarked.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.CaptureCalls != 2 {
		t.Fatalf("CaptureCalls=%d want 2 (inventory + notes lastSeen)", resp.CaptureCalls)
	}
	if !resp.FromCache {
		t.Fatal("second GET should hit cache")
	}
	if !resp.FirstBookmarked || resp.BookmarkCount != 1 {
		t.Fatalf("bookmarked=%v count=%d", resp.FirstBookmarked, resp.BookmarkCount)
	}
}
```
