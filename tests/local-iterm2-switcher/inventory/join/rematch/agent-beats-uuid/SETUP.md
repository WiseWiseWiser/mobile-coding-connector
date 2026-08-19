# Scenario

**Feature**: agent pair match wins over a UUID-only item on the same pane

```
item A UUID sess-a + item B grok/g1 sess-old -> live sess-a Agent grok/g1
BuildInventory -> live note is agent-item; uuid-item is orphan
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
	req.NotesJSON = `{"version":2,"items":[{"note":"uuid-item","bookmarked":true,"iterm_session_id":"sess-a"},{"note":"agent-item","bookmarked":true,"agent_runner":"grok","grok_session_id":"g1","iterm_session_id":"sess-old"}]}`
	return nil
}
```
