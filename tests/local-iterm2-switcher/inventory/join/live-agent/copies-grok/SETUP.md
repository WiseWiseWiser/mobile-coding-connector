# Scenario

**Feature**: grok Agent on the snap is copied onto the live row JSON

```
sess-a Agent grok/g1 -> BuildInventory
live agent_runner=grok grok_session_id=g1
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
	return nil
}
```
