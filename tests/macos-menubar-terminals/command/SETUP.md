# Scenario

**Feature**: attach/new CLI builders and agent binary selection

```
agentBinary + sessionID -> BuildTerminalAttachCommand / BuildTerminalNewCommand
isRemote -> AgentBinaryForApp -> local-agent | remote-agent
```

## Preconditions

`Op` is one of `attach_cmd`, `new_cmd`, `agent_binary` set by each leaf.

## Steps

1. Leaf sets op-specific inputs (`AgentBinary`, `SessionID`, `IsRemote`).

## Context

REQUIREMENT: local uses `local-agent`, remote uses `remote-agent`; prefer session id in attach.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// Leaves set Op and command-builder inputs.
	if req == nil {
		t.Fatal("nil request")
	}
	return nil
}
```
