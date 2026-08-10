# Scenario

**Feature**: send success prints message id

```
seed live + SendInject=ok -> agent-run send sess-send "hi" -> msg_1
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "send", "sess-send", "hello follow-up")
	req.Seeds = seedLiveTTY("sess-send", "term-send")
	req.SendInject = "ok"
	req.SendResultMsgID = "msg_1"
	return nil
}
```
