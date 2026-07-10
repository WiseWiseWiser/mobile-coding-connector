# Scenario

**Feature**: cron enable action path encodes id

```
CronActionPath(enable, "task-1") -> "/api/cron-tasks/enable?id=task-1" (URL-encoded as needed)
```

## Preconditions

POST control path for `enable` with query id.

## Steps

1. Set leaf, action, and TaskID.

## Context

REQUIREMENT leaf: `cronapi/enable-path`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.CronAPILeaf = "enable-path"
	req.CronAction = "enable"
	req.TaskID = "task-1"
	return nil
}
```
