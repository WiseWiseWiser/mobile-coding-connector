# Scenario

**Feature**: send fails when TTY unreachable

```
SendInject=error-unreachable -> agent-run send sess-dead "x" -> error
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "send", "sess-dead", "ping")
	req.Seeds = seedLiveTTY("sess-dead", "term-dead")
	req.SendInject = "error-unreachable"
	return nil
}
```
