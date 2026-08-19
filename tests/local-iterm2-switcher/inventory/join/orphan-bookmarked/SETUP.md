# Scenario

**Feature**: starred-but-gone session is a saved_notes orphan

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "join"
	req.ITermRunning = true
	req.WindowSpace = 1
	req.NotesJSON = `{"version":1,"notes":{"sess-dead":{"note":"","bookmarked":true,"last_seen":{"cwd":"/tmp/gone","space_index":1}}}}`
	return nil
}
```
