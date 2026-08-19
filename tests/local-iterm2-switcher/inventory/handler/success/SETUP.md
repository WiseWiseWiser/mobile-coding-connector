# Scenario

**Feature**: GET inventory 200 with live session

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "inventory"
	req.ITermRunning = true
	req.WindowSpace = 1
	req.NotesJSON = `{"version":1,"notes":{"sess-a":{"note":"fix auth"}}}`
	return nil
}
```
