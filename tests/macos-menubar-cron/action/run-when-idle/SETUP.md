# Scenario

**Feature**: Run Now enabled when task is idle

```
CanRunCronTask("idle") -> true
```

## Preconditions

Task status is `idle`.

## Steps

1. Set `Status=idle`.

## Context

REQUIREMENT leaf: `action/run-when-idle`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Status = "idle"
	return nil
}
```
