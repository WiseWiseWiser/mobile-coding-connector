# Scenario

**Feature**: local new-terminal command line

```
BuildTerminalNewCommand("local-agent") -> "local-agent terminal new"
```

## Preconditions

New Terminal… runs without prompt; local-agent binary.

## Steps

1. Set `Op=new_cmd`, binary `local-agent`.

## Context

REQUIREMENT leaf: `command/new-local`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "new_cmd"
	req.AgentBinary = "local-agent"
	return nil
}
```
