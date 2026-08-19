## Expected

1. FocusPane is sidebar after Tab.
2. View shows sidebar cursor (`›` near All/Bookmarks/Desktop) and list content still present.

```go
import (
	"regexp"
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.FocusPane != "sidebar" {
		t.Fatalf("FocusPane=%q want sidebar after tab", resp.FocusPane)
	}
	if resp.ViewText == "" {
		t.Fatal("View empty after tab")
	}
	if !resp.HasSidebarCursor && !regexp.MustCompile(`›`).MatchString(resp.ViewText) {
		t.Fatalf("sidebar focus should show › on a left row; view=%q", resp.ViewText)
	}
	if !regexp.MustCompile(`grok review|All|Desktop`).MatchString(resp.ViewText) {
		t.Fatalf("list/sidebar content still visible; view=%q", resp.ViewText)
	}
}
```
