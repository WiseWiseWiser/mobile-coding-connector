# Scenario

**Feature**: omitting timeout on create defaults to 1h

```
# POST create without timeout field
definition/status Timeout == "1h" (or equivalent duration string)
```

## Preconditions

1. Create body omits `timeout`.
2. Interval long (`1h`) so we do not wait for fires.

## Steps

1. Create interval task without timeout.
2. List/status must show default `1h`.

## Context

Priority leaf: default timeout 1h.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Action = "create"
	req.TaskName = "default-timeout"
	req.Command = "echo ok"
	req.ScheduleMode = "interval"
	req.Interval = "1h"
	// Timeout intentionally empty → omit field
	req.Timeout = ""
	return nil
}
```
