# Scenario

**Feature**: with fake opencode on PATH, grok and opencode report installed

```
fake opencode binary on PATH -> isAgentInstalled -> both true
```

## Preconditions

- Fake opencode built from testdata and prepended to PATH.

## Steps

1. `UseFakeOpenCode = true`, `Op = OpListAgents`.

## Context

Install parity when binary is resolvable.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpListAgents
	req.UseFakeOpenCode = true
	return nil
}
```