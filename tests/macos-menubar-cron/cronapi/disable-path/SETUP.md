# Scenario

**Feature**: cron disable action path encodes id

```
CronActionPath(disable, "task-1") -> "/api/cron-tasks/disable?id=task-1" (URL-encoded as needed)
```

## Preconditions

POST control path for `disable` with query id.

## Steps

1. Set leaf, action, and TaskID.

## Context

REQUIREMENT leaf: `cronapi/disable-path`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.CronAPILeaf = "disable-path"
	req.CronAction = "disable"
	req.TaskID = "task-1"
	return nil
}
```
