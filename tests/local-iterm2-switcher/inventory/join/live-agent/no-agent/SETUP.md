# Scenario

**Feature**: snap without Agent leaves live agent fields empty

```
item grok/g1 on sess-a + live sess-a (no Agent) -> BuildInventory
live bookmarked via UUID; agent_runner and grok_session_id empty
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
	req.NotesJSON = `{"version":2,"items":[{"note":"staging fix","bookmarked":true,"agent_runner":"grok","grok_session_id":"g1","iterm_session_id":"sess-a"}]}`
	return nil
}
```
