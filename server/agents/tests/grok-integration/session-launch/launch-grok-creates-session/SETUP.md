# Scenario

**Feature**: launch grok creates a headless agent session

```
launch(grok, project_dir) -> opencode serve --port -> session agent_id=grok
```

## Preconditions

- Fake opencode on PATH that serves /global/health on allocated port.

## Steps

1. `UseFakeOpenCode = true`, `Op = OpLaunchGrok`.

## Context

Happy path for POST /api/agents/sessions with agent_id grok.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpLaunchGrok
	req.UseFakeOpenCode = true
	return nil
}
```