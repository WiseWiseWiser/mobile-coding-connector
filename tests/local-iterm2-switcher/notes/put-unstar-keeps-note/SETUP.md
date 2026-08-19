# Scenario

**Feature**: unstar leaves the note in place

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
	req.Bookmarked = false
	req.NotesJSON = `{"version":1,"notes":{"sess-a":{"note":"fix auth on staging","bookmarked":true}}}`
	return nil
}
```
