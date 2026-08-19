## Expected

1. After `j`, list cursor is on the next session (not sess-a / grok review).
2. View still shows multiple sessions or at least the second primary name.

```go
import (
	"regexp"
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.ViewText == "" {
		t.Fatal("View empty after j (TUI not implemented)")
	}
	if resp.ListCursorSession == "sess-a" || resp.ListCursorSession == "" {
		t.Fatalf("after j, › must move off first session; ListCursorSession=%q view=%q",
			resp.ListCursorSession, resp.ViewText)
	}
	// second pane on Desktop 1 in two-session fixture
	if resp.ListCursorSession != "sess-a2" && !regexp.MustCompile(`second pane`).MatchString(resp.ViewText) {
		t.Fatalf("want cursor on second session (sess-a2 / second pane); got %q view=%q",
			resp.ListCursorSession, resp.ViewText)
	}
}
```
