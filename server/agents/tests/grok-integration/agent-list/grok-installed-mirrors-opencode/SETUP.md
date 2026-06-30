# Scenario

**Feature**: Grok installed flag mirrors OpenCode binary resolution

```
# isAgentInstalled(grok) uses same opencode path rules as opencode agent
Agent catalog -> grok.installed == opencode.installed
```

## Preconditions

- PATH controlled per leaf: stripped vs fake opencode binary.

## Steps

1. Child sets `StripOpenCode` or `UseFakeOpenCode`.
2. `Request.Op = OpListAgents`.

## Context

MECE on opencode availability: both absent vs both present on PATH.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpListAgents
	return nil
}
```