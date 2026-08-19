# Scenario

**Feature**: session on Desktop 2 with joined note

```
snap window 42 space=1 + note sess-a -> Desktop 2 row has note
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "join"
	req.ITermRunning = true
	req.WindowSpace = 1
	req.NotesJSON = `{"version":1,"notes":{"sess-a":{"note":"fix auth on staging"}}}`
	return nil
}
```
