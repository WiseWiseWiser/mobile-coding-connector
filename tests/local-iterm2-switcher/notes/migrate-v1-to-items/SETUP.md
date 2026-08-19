# Scenario

**Feature**: load v1 notes map migrates to items with iterm_session_id only

```
v1 notes{sess-a} file -> NoteStore.Document
items[0].iterm_session_id=sess-a; no agent fields
BuildInventory -> live sess-a bookmarked
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "notes_store"
	req.ITermRunning = true
	req.WindowSpace = 1
	req.SessionID = "sess-a"
	req.NotesJSON = `{"version":1,"notes":{"sess-a":{"note":"staging fix","bookmarked":true}}}`
	return nil
}
```
