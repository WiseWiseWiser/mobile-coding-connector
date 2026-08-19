## Expected

1. Exit 0.
2. FocusCalled with FocusSession=sess-a.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit=%d out=%q err=%q", resp.ExitCode, resp.Combined, resp.ErrMsg)
	}
	if !resp.FocusCalled {
		t.Fatal("Enter must call Focus")
	}
	if resp.FocusSession != "sess-a" {
		t.Fatalf("FocusSession=%q want sess-a", resp.FocusSession)
	}
}
```
