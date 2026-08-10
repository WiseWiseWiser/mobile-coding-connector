# Scenario

**Feature**: agent-run attach help documents session-id

```
remote-agent agent-run attach --help -> Usage ... session-id ...
```

## Preconditions

None beyond L2 harness.

## Steps

1. Args = `agent-run attach --help`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "attach", "--help")
	return nil
}
```
