# Scenario

**Feature**: note for a dead session UUID becomes saved_notes

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "join"
	req.ITermRunning = true
	req.WindowSpace = 1
	req.NotesJSON = `{"version":1,"notes":{"sess-dead":{"note":"machine backup plan","last_seen":{"cwd":"/Users/xhd2015/wrk/backup","space_index":1}}}}`
	return nil
}
```
