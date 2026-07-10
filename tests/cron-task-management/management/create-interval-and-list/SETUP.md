# Scenario

**Feature**: create interval task via API then list shows it

```
# POST /api/cron-tasks interval definition
create -> GET /api/cron-tasks includes name, command, scheduleMode=interval
```

## Preconditions

1. Create body: name, command, scheduleMode=interval, interval=5m.
2. Timeout omitted (default applied separately in default-timeout-1h leaf).

## Steps

1. Action `create` with interval fields.
2. Run lists after create; Assert finds the task.

## Context

CRUD priority: create + list.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Action = "create"
	req.TaskName = "echo-every-5m"
	req.Command = "echo hello-cron"
	req.ScheduleMode = "interval"
	req.Interval = "5m"
	return nil
}
```
