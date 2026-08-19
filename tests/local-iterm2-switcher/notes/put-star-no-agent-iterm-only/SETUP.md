# Scenario

**Feature**: PUT star with no live agent persists iterm_session_id only

```
fixture sess-a (no Agent) -> PUT bookmarked
NoteStore item has iterm_session_id only
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "notes"
	req.ITermRunning = true
	req.SessionID = "sess-a"
	req.OmitNote = true
	req.SetBookmarked = true
	req.Bookmarked = true
	return nil
}
```
