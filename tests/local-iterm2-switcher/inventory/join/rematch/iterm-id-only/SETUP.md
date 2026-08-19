# Scenario

**Feature**: item with only iterm_session_id joins by UUID

```
item iterm sess-a (no agent fields) -> live sess-a Agent grok/g1
BuildInventory -> live row bookmarked (UUID only)
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
	req.AgentKind = "grok"
	req.AgentSessionID = "g1"
	req.NotesJSON = `{"version":2,"items":[{"note":"staging fix","bookmarked":true,"iterm_session_id":"sess-a"}]}`
	return nil
}
```
