# Scenario

**Feature**: empty server message falls back to client copy

```
CronToggleAlertMessage("") / whitespace -> "Task updated"
```

## Preconditions

Server returned CronTaskStatus without a message field (current API shape).

## Steps

1. Set empty `ServerMessage`.

## Context

REQUIREMENT leaf: `alert/empty-uses-fallback`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ServerMessage = ""
	return nil
}
```
