# Scenario

**Feature**: PUT empty note deletes the entry

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "notes"
	req.ITermRunning = true
	req.SessionID = "sess-a"
	req.Note = ""
	req.NotesJSON = `{"version":1,"notes":{"sess-a":{"note":"old note"}}}`
	return nil
}
```
