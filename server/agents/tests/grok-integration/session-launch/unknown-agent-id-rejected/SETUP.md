# Scenario

**Feature**: unknown agent id rejected at launch

```
launch(not-a-real-agent) -> unknown agent error
```

## Preconditions

- Valid project directory.

## Steps

1. `UnknownAgentID = true`, `Op = OpLaunchGrok`.

## Context

Maps to HTTP 400 unknown agent for API layer.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpLaunchGrok
	req.UnknownAgentID = true
	return nil
}
```