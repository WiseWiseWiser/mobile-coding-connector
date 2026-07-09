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
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "agent_binary"
	req.IsRemote = true
	return nil
}
```
