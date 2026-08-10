# Scenario

**Feature**: run --detach success via inject

```
RunInject=ok -> agent-run run --detach --session-id s1 "hi"
  -> exit 0, prints session/terminal ids
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "run", "--detach", "--session-id", "sess-run-1", "hello remote")
	req.RunInject = "ok"
	req.RunResultSessionID = "sess-run-1"
	req.RunResultTerminalID = "term-run-1"
	return nil
}
```
