# Scenario

**Feature**: Delete allowed when task is in error state

```
CanDeleteCronTask("error") -> true
```

## Preconditions

Task status is `error` (not running).

## Steps

1. Set `Status=error`.

## Context

REQUIREMENT leaf: `delete-gate/when-error` (scenario 3).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Status = "error"
	return nil
}
```
