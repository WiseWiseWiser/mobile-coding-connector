# Scenario

**Feature**: prefer non-empty server message for toggle alert

```
CronToggleAlertMessage("Task disabled until next schedule") -> same string
```

## Preconditions

Enable/disable API (or client) surfaces a non-empty message.

## Steps

1. Set a non-empty `ServerMessage`.

## Context

REQUIREMENT leaf: `alert/prefer-server-message`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ServerMessage = "Task disabled until next schedule"
	return nil
}
```
