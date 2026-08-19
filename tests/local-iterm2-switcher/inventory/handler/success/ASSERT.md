## Expected

1. HTTP 200.
2. `iterm_running` true, one session, note joined.

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
		t.Fatalf("status=%d body=%s", resp.StatusCode, resp.Body)
	}
	if !resp.ITermRunning || resp.SessionCount != 1 {
		t.Fatalf("running=%v sessions=%d", resp.ITermRunning, resp.SessionCount)
	}
	if resp.NoteOnFirst != "fix auth" {
		t.Fatalf("note=%q", resp.NoteOnFirst)
	}
}
```
