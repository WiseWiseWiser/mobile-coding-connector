# Scenario

**Feature**: v2 bookmark items rematch onto live panes

```
v2 items + fixture snap Agent -> BuildInventory
# agent pair first, then iterm_session_id
miss -> saved_notes
```

## Preconditions

iTerm is running. Fixture live pane is `sess-a`. NotesJSON is a v2 `items` list.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "join"
	req.ITermRunning = true
	req.WindowSpace = 1
	return nil
}
```
