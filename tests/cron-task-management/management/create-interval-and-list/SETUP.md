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
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Action = "create"
	req.TaskName = "echo-every-5m"
	req.Command = "echo hello-cron"
	req.ScheduleMode = "interval"
	req.Interval = "5m"
	req.UseBinary = true // L3 smoke
	return nil
}
```
