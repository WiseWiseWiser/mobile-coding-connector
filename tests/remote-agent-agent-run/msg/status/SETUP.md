# Scenario

**Feature**: msg status pending inject

```
MsgStatusInject=pending -> agent-run msg status sess-m/msg_1 -> pending
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "msg", "status", "sess-m/msg_1")
	req.Seeds = seedLiveTTY("sess-m", "term-m")
	req.MsgStatusInject = "pending"
	return nil
}
```
