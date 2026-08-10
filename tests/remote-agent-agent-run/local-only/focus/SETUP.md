# Scenario

**Feature**: focus is local-only

```
agent-run focus … -> Error: not available via remote-agent (local-only)
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "focus")
	return nil
}
```
