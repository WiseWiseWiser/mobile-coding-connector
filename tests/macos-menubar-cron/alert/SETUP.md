# Scenario

**Feature**: enable/disable NSAlert message (server or fallback)

```
serverMessage -> CronToggleAlertMessage -> alert body; empty -> "Task updated"
```

## Preconditions

`Op=alert` dispatches to `menubar.CronToggleAlertMessage`.

## Steps

1. Leaf supplies `ServerMessage`.

## Context

REQUIREMENT: show alert with server message (or fallback), then refresh.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "alert"
	return nil
}
```
