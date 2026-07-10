# Scenario

**Feature**: reject create when neither interval nor cron schedule is provided

```
# body with name+command only (no scheduleMode / interval / cronExpr)
POST -> non-2xx error
```

## Preconditions

1. Raw body omits schedule fields entirely.

## Steps

1. POST incomplete body.
2. Assert error; task `no-schedule` absent.

## Context

Validation: schedule required.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Action = "create"
	req.RawBody = map[string]any{
		"name":    "no-schedule",
		"command": "echo no",
		"timeout": "1h",
	}
	return nil
}
```
