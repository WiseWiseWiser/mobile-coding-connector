# Scenario

**Feature**: launch grok creates a headless agent session

```
launch(grok, project_dir) -> opencode serve --port -> session agent_id=grok
```

## Preconditions

- Prefer **real** `opencode` in PATH when installed; fake opencode only as fallback when absent.
- Isolated `AI_CRITIC_HOME` per run (set in root `Run`).

## Steps

1. `Op = OpLaunchGrok` (do not force `UseFakeOpenCode`).

## Context

Happy path for POST /api/agents/sessions with agent_id grok; label `slow` when real binary starts.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpLaunchGrok
	return nil
}
```