# Scenario

**Feature**: update cron task request is PUT with id in JSON body

```
BuildUpdateCronTaskRequest(base, token, def with id)
  -> PUT https://agent.example.com/api/cron-tasks
  -> body includes id + name/command/schedule
```

## Preconditions

1. Update uses same collection path as create; method is PUT.
2. Definition `id` is required in body (not only query).
3. `cronExpr` remains UTC when sent.

## Steps

1. Set update leaf, base URL, TaskID, and definition fields.

## Context

REQUIREMENT leaf: `cronapi/update-path`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.CronAPILeaf = "update-path"
	req.BaseURL = "https://agent.example.com/"
	req.Token = "secret-token"
	req.TaskID = "task-1"
	req.Name = "backup"
	req.Command = "echo backup"
	req.ScheduleMode = "interval"
	req.Interval = "5m"
	req.Timeout = "1h"
	en := true
	req.Enabled = &en
	return nil
}
```
