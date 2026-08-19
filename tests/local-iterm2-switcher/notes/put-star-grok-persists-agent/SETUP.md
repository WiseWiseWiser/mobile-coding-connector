# Scenario

**Feature**: PUT star with live grok agent persists runner + grok + iTerm ids

```
fixture sess-a Agent grok/g1 -> PUT bookmarked
NoteStore item has agent_runner, grok_session_id, iterm_session_id
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "notes"
	req.ITermRunning = true
	req.SessionID = "sess-a"
	req.OmitNote = true
	req.SetBookmarked = true
	req.Bookmarked = true
	req.AgentKind = "grok"
	req.AgentSessionID = "g1"
	return nil
}
```
