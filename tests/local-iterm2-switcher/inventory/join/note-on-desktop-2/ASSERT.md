## Expected

1. `ITermRunning` true.
2. Two Desktops; one live session.
3. Session is on Desktop 2 (space index 1).
4. Joined note is `fix auth on staging`.
5. No orphans.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ITermRunning {
		t.Fatal("want iterm running")
	}
	if resp.DesktopCount != 2 {
		t.Fatalf("DesktopCount=%d want 2", resp.DesktopCount)
	}
	if resp.SessionCount != 1 {
		t.Fatalf("SessionCount=%d want 1", resp.SessionCount)
	}
	if resp.FirstDesktop != 2 || resp.FirstSpace != 1 {
		t.Fatalf("session desktop=%d space=%d want 2/1", resp.FirstDesktop, resp.FirstSpace)
	}
	if resp.NoteOnFirst != "fix auth on staging" {
		t.Fatalf("note=%q", resp.NoteOnFirst)
	}
	if resp.HasOrphan {
		t.Fatal("unexpected orphan")
	}
}
```
