# Scenario

**Feature**: --new-terminal rejected on remote-agent

```
agent-run run --new-terminal --detach "x" -> Error: not available / remote
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "run", "--new-terminal", "--detach", "hello")
	// Inject unused if parser rejects before server call.
	req.RunInject = "ok"
	return nil
}
```
