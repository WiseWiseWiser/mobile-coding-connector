# Scenario

**Feature**: kill --dry-run reports without terminate

```
KillInject=ok -> agent-run kill --dry-run sess-k -> exit 0, dry-run signal
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "kill", "--dry-run", "sess-k")
	req.Seeds = seedLiveTTY("sess-k", "term-k")
	req.KillInject = "ok"
	return nil
}
```
