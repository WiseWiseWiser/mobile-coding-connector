```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasClickOutsideDismiss {
		t.Fatal("switcher must dismiss on mouse-down outside the panel (local+global monitors)")
	}
}
```
