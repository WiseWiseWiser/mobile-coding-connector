# Scenario

**Feature**: launch grok fails when opencode is not installed

```
PATH without opencode -> getAgentBinaryPath fails -> launch error
```

## Preconditions

- Stripped PATH.

## Steps

1. `StripOpenCode = true`, `Op = OpLaunchGrok`.

## Context

Error message should mention install or opencode (implementer).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpLaunchGrok
	req.StripOpenCode = true
	return nil
}
```