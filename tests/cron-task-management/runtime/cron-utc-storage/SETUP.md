# Scenario

**Feature**: cron expression stored and treated as UTC on server

```
# POST create scheduleMode=cron cronExpr="0 9 * * *"
list shows cronExpr exactly "0 9 * * *" (UTC stored form)
```

## Preconditions

1. API create uses UTC expression (equivalent to CLI `--cron-utc`).
2. No local conversion on server.

## Steps

1. Create cron-mode task with expr `0 9 * * *`.
2. Assert list stores same expression and scheduleMode=cron.

## Context

Priority leaf: Cron UTC storage. First fire timing is not asserted (may be hours away).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Action = "create"
	req.TaskName = "morning-utc"
	req.Command = "echo utc-cron"
	req.ScheduleMode = "cron"
	req.CronExpr = "0 9 * * *"
	req.Timeout = "1h"
	return nil
}
```
