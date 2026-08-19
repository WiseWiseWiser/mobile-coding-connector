```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.SidebarTitle != "Desktop 2" {
		t.Fatalf("title=%q", resp.SidebarTitle)
	}
	if resp.DesktopHeader != "Desktop 2" {
		t.Fatalf("header=%q", resp.DesktopHeader)
	}
}
```
