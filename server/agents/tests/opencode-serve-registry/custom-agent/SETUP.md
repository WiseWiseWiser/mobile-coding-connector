# Scenario

**Feature**: custom agent launch registry (Path B)

```
LaunchCustomAgent -> opencode serve -> kind=custom-agent in registry
```

## Preconditions

- Custom agent fixture under `$HOME/.ai-critic/agents/<id>/agent.json`.
- Fake opencode on PATH.

## Steps

1. Set `Op = OpCustomRegistry` in leaf.

## Context

Path B uses separate custom agent config dir but same children registry file.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.UseFakeOpenCode = true
	return nil
}
```
