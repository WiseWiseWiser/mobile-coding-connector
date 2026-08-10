# Scenario

**Feature**: msg cancel success inject

```
MsgCancelInject=ok -> agent-run msg cancel sess-m/msg_1 -> exit 0
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "msg", "cancel", "sess-m/msg_1")
	req.Seeds = seedLiveTTY("sess-m", "term-m")
	req.MsgCancelInject = "ok"
	return nil
}
```
