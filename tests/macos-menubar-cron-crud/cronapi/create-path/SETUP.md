# Scenario

**Feature**: create cron task request is POST with JSON definition body

```
BuildCreateCronTaskRequest(base, token, def)
  -> POST https://agent.example.com/api/cron-tasks
  -> body: name, command, scheduleMode=cron, cronExpr=UTC, timeout=1h
```

## Preconditions

1. Create uses collection path `/api/cron-tasks` (no id in path).
2. Body is JSON; `cronExpr` is already UTC when built (convert happens before).
3. Optional Bearer when token set.

## Steps

1. Set create leaf, base URL, definition fields (UTC cronExpr).

## Context

REQUIREMENT leaf: `cronapi/create-path`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.CronAPILeaf = "create-path"
	req.BaseURL = "https://agent.example.com/"
	req.Token = "secret-token"
	req.Name = "backup"
	req.Command = "echo backup"
	req.ScheduleMode = "cron"
	req.CronExpr = "0 1 * * *" // UTC as stored/sent
	req.Timeout = "1h"
	en := true
	req.Enabled = &en
	return nil
}
```
