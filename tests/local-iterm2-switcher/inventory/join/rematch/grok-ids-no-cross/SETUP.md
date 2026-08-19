# Scenario

**Feature**: two different grok ids do not cross-join

```
item grok/g1 + iterm sess-old -> live sess-a Agent grok/g2
BuildInventory -> live unstarred; item is orphan
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
	req.AgentSessionID = "g2"
	req.NotesJSON = `{"version":2,"items":[{"note":"staging fix","bookmarked":true,"agent_runner":"grok","grok_session_id":"g1","iterm_session_id":"sess-old"}]}`
	return nil
}
```
