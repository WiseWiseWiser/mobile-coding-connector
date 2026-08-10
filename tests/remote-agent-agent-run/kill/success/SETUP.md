# Scenario

**Feature**: kill success inject stops session

```
KillInject=ok -> agent-run kill sess-k -> exit 0
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "kill", "sess-k")
	req.Seeds = seedLiveTTY("sess-k", "term-k")
	req.KillInject = "ok"
	return nil
}
```
