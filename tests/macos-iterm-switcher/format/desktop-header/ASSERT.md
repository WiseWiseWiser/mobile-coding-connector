```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.DesktopHeader != "Desktop 3" {
		t.Fatalf("header=%q", resp.DesktopHeader)
	}
}
```
