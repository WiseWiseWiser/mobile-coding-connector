# Scenario

**Feature**: send without session id or message fails

```
agent-run send -> Error: requires session-id and message
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "send")
	return nil
}
```
