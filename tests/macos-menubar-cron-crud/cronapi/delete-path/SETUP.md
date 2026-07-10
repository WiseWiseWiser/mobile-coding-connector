# Scenario

**Feature**: delete cron task request is DELETE with id query

```
BuildDeleteCronTaskRequest(base, token, "task-1")
  -> DELETE https://agent.example.com/api/cron-tasks?id=task-1
```

## Preconditions

1. Delete uses query `id` (not JSON body).
2. Path helper encodes id for special characters when needed.

## Steps

1. Set delete leaf, base URL, TaskID.

## Context

REQUIREMENT leaf: `cronapi/delete-path`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.CronAPILeaf = "delete-path"
	req.BaseURL = "https://agent.example.com/"
	req.Token = "secret-token"
	req.TaskID = "task-1"
	return nil
}
```
