## Expected

1. HTTP 200 `ok:true`.
2. Focus session sess-a tab 2 window 42. Switch is not called.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || !resp.OK {
		t.Fatalf("status=%d ok=%v body=%s", resp.StatusCode, resp.OK, resp.Body)
	}
	if resp.SwitchCalled {
		t.Fatalf("Switch must not run on focus")
	}
	if !resp.FocusCalled || resp.FocusSession != "sess-a" || resp.FocusTab != 2 || resp.FocusWindow != "42" {
		t.Fatalf("focus sess=%q tab=%d win=%q", resp.FocusSession, resp.FocusTab, resp.FocusWindow)
	}
}
```
