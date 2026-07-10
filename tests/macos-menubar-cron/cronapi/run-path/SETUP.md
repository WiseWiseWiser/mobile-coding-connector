# Scenario

**Feature**: cron run action path encodes id

```
CronActionPath(run, "task-1") -> "/api/cron-tasks/run?id=task-1" (URL-encoded as needed)
```

## Preconditions

POST control path for `run` with query id.

## Steps

1. Set leaf, action, and TaskID.

## Context

REQUIREMENT leaf: `cronapi/run-path`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.CronAPILeaf = "run-path"
	req.CronAction = "run"
	req.TaskID = "task-1"
	return nil
}
```
