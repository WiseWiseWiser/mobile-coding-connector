# Scenario

**Feature**: enabled cron task shows Disable action

```
ShowEnableCronAction(true) -> false (show Disable, not Enable)
```

## Preconditions

Task definition has `enabled=true`.

## Steps

1. Set `Enabled=true`.

## Context

REQUIREMENT leaf: `action/show-disable`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Enabled = true
	return nil
}
```
