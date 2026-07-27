# Scenario

**Feature**: agent binary for local app

```
AgentBinaryForApp(false) -> "local-agent"
```

## Preconditions

Local menu-bar app profile (`isRemote=false`).

## Steps

1. Set `Op=agent_binary`, `IsRemote=false`.

## Context

REQUIREMENT leaf: `command/agent-binary` (local branch).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "agent_binary"
	req.IsRemote = false
	return nil
}
```
