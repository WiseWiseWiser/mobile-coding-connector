## Expected

1. `ITermRunning` false.
2. Desktops still listed (2).
3. Zero live sessions.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ITermRunning {
		t.Fatal("want iterm not running")
	}
	if resp.DesktopCount != 2 {
		t.Fatalf("DesktopCount=%d want 2", resp.DesktopCount)
	}
	if resp.SessionCount != 0 {
		t.Fatalf("SessionCount=%d want 0", resp.SessionCount)
	}
}
```
