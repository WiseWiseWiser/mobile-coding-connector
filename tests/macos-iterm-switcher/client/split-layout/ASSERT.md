```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasNativeSplit {
		t.Fatal("switcher must use NavigationSplitView + searchable")
	}
	if !resp.HasBookmarkAction {
		t.Fatal("switcher must expose bookmark toggle")
	}
	if !resp.HasDoubleClickFocus {
		t.Fatal("switcher must focus on double-click")
	}
}
```
