## Expected

1. HTTP 200 (not 500).
2. `iterm_running` false, zero sessions, Desktops present.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d want 200 body=%s", resp.StatusCode, resp.Body)
	}
	if resp.ITermRunning {
		t.Fatal("want iterm_running false")
	}
	if resp.SessionCount != 0 {
		t.Fatalf("sessions=%d", resp.SessionCount)
	}
	if resp.DesktopCount < 1 {
		t.Fatal("want listed desktops")
	}
}
```
