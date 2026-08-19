# Scenario

**Feature**: agent_runner without grok_session_id does not agent-match

```
item runner=grok (no grok id) + iterm sess-old -> live sess-a Agent grok/g1
BuildInventory -> no agent hit; UUID miss; orphan
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
	req.NotesJSON = `{"version":2,"items":[{"note":"staging fix","bookmarked":true,"agent_runner":"grok","iterm_session_id":"sess-old"}]}`
	return nil
}
```
