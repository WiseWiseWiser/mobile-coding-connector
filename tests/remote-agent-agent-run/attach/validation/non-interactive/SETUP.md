# Scenario

**Feature**: attach refuses non-interactive stdin/stdout

```
RunWithWriters (pipe stdout) + agent-run attach <id>
  -> Error: requires interactive terminal
```

## Preconditions

- L2 CLI uses injected writers so stdout is not a TTY (same gate as
  `terminal attach`).
- Session id is present so the failure is the TTY gate, not arg parse.
- Store may be empty; product may check TTY before network.

## Steps

1. Args = `agent-run attach sess-any`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// Valid-looking id so arg validation passes; TTY check should fail under pipes.
	setCLI(req, "agent-run", "attach", "sess-any")
	return nil
}
```
