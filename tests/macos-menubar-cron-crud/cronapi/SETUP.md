# Scenario

**Feature**: pure cron CRUD request builders (create / update / delete)

```
# paths
CreateCronTasksPath / UpdateCronTasksPath -> "/api/cron-tasks"
DeleteCronTaskPath(id) -> "/api/cron-tasks?id="

# requests
BuildCreateCronTaskRequest(base, token, def) -> POST + JSON body + optional Bearer
BuildUpdateCronTaskRequest(base, token, def) -> PUT + JSON body (id required)
BuildDeleteCronTaskRequest(base, token, id)  -> DELETE + query id
```

## Preconditions

`Op=cronapi` dispatches to `macosapp/cronapi`. Extends list/action builders;
`cronExpr` in body is always UTC; UI omits `extraEnv`.

## Steps

1. Leaf sets `CronAPILeaf` and BaseURL/Token/TaskID/definition fields.

## Context

REQUIREMENT: create POST, update PUT, delete DELETE paths + body method;
delete requires id.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "cronapi"
	return nil
}
```
