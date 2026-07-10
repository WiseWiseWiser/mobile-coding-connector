# Scenario

**Feature**: disabled cron task shows Enable action

```
ShowEnableCronAction(false) -> true (show Enable, not Disable)
```

## Preconditions

Task definition has `enabled=false`.

## Steps

1. Set `Enabled=false`.

## Context

REQUIREMENT leaf: `action/show-enable`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Enabled = false
	return nil
}
```
