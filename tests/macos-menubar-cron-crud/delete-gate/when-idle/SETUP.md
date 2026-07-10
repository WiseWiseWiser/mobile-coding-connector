# Scenario

**Feature**: Delete allowed when task is idle

```
CanDeleteCronTask("idle") -> true
```

## Preconditions

Task status is `idle` (not running).

## Steps

1. Set `Status=idle`.

## Context

REQUIREMENT leaf: `delete-gate/when-idle` (scenario 3).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Status = "idle"
	return nil
}
```
