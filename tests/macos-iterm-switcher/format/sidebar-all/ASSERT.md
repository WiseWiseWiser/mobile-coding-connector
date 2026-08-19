```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.SidebarTitle != "All" {
		t.Fatalf("title=%q", resp.SidebarTitle)
	}
	if resp.WindowTitle != "Terminals" {
		t.Fatalf("window=%q", resp.WindowTitle)
	}
	if resp.FilteredCount != 3 {
		t.Fatalf("filtered=%d want 3", resp.FilteredCount)
	}
}
```
