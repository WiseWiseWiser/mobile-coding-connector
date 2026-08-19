# Scenario

**Feature**: non-grok Agent copies kind only

```
sess-a Agent codex/c1 -> BuildInventory
live agent_runner=codex; grok_session_id empty
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
	req.AgentKind = "codex"
	req.AgentSessionID = "c1"
	return nil
}
```
