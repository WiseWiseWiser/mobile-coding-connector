## Expected

1. Label exactly `No bookmarks`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.Label != "No bookmarks" {
		t.Fatalf("label=%q want No bookmarks", resp.Label)
	}
}
```
