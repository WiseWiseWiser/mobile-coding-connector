## Expected

1. Switch is not called.
2. Focus still runs.
3. HTTP 200.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.SwitchCalled {
		t.Fatal("Switch must not run on focus")
	}
	if !resp.FocusCalled {
		t.Fatal("Focus must run")
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d want 200 body=%s", resp.StatusCode, resp.Body)
	}
}
```
