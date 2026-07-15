# Scenario

**Feature**: sessionMgr.launch for agent grok

```
Session manager -> opencode serve -> health -> session running
```

## Preconditions

- Project dir is a real directory unless invalid-project leaf.

## Steps

1. Set `Op = OpLaunchGrok` and leaf-specific flags.

## Context

MECE: success with real opencode (fake fallback), registry cleanup, missing binary, unknown id, invalid dir.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpLaunchGrok
	return nil
}
```