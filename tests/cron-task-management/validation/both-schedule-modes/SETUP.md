# Scenario

**Feature**: reject create when both interval and cron schedule fields are set

```
# body scheduleMode ambiguous or both interval + cronExpr present
POST -> non-2xx error
```

## Preconditions

1. Raw body includes name, command, interval, and cronExpr together.
2. May also set scheduleMode to one of them; still invalid when both schedule values present.

## Steps

1. POST RawBody with both `interval` and `cronExpr`.
2. Assert error status and no successful task named `both-modes`.

## Context

Validation: XOR schedule.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Action = "create"
	req.RawBody = map[string]any{
		"name":         "both-modes",
		"command":      "echo no",
		"scheduleMode": "interval",
		"interval":     "5m",
		"cronExpr":     "0 9 * * *",
		"timeout":      "1h",
	}
	return nil
}
```
