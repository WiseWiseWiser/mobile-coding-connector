# Scenario

**Feature**: agent binary for remote app

```
AgentBinaryForApp(true) -> "remote-agent"
```

## Preconditions

Remote menu-bar app profile (`isRemote=true`).

## Steps

1. Set `Op=agent_binary`, `IsRemote=true`.

## Context

REQUIREMENT leaf: `command/agent-binary` (remote branch).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "agent_binary"
	req.IsRemote = true
	return nil
}
```
