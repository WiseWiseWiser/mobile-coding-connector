# Scenario

**Feature**: live inventory JSON carries snap agent fields

```
fixture sess-a Agent -> BuildInventory
live row JSON agent_runner / grok_session_id
```

## Preconditions

iTerm is running. Fixture live pane is `sess-a`. Agent is injected on the snap (no procresolve).

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
