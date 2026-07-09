# Scenario

**Feature**: remote new-terminal command line

```
BuildTerminalNewCommand("remote-agent") -> "remote-agent terminal new"
```

## Preconditions

New Terminal… for remote app uses remote-agent.

## Steps

1. Set `Op=new_cmd`, binary `remote-agent`.

## Context

REQUIREMENT leaf: `command/new-remote`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "new_cmd"
	req.AgentBinary = "remote-agent"
	return nil
}
```
