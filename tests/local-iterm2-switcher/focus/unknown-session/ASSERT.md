## Expected

1. HTTP 404, error session not found.
2. Switch and Focus not called.

```go
import (
	"strings"
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 404 {
		t.Fatalf("status=%d want 404 body=%s", resp.StatusCode, resp.Body)
	}
	if !strings.Contains(resp.Error, "not found") {
		t.Fatalf("error=%q", resp.Error)
	}
	if resp.SwitchCalled || resp.FocusCalled {
		t.Fatal("Switch/Focus must not run for unknown id")
	}
}
```
