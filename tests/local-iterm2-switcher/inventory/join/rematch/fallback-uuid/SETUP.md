# Scenario

**Feature**: grok id not live falls back to iterm_session_id

```
item grok/g-missing + iterm sess-a -> live sess-a Agent grok/g-other
BuildInventory -> live row bookmarked (UUID fallback)
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
	req.AgentSessionID = "g-other"
	req.NotesJSON = `{"version":2,"items":[{"note":"staging fix","bookmarked":true,"agent_runner":"grok","grok_session_id":"g-missing","iterm_session_id":"sess-a"}]}`
	return nil
}
```
