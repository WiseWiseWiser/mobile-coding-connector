## Expected

1. Live session is not bookmarked.
2. One orphan for the gone starred session.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.FirstBookmarked {
		t.Fatal("live session should not be bookmarked")
	}
	if !resp.HasOrphan || resp.SavedCount != 1 {
		t.Fatalf("SavedCount=%d HasOrphan=%v", resp.SavedCount, resp.HasOrphan)
	}
}
```
