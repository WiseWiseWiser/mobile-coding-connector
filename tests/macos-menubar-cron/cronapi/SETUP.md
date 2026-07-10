# Scenario

**Feature**: pure cron API path and request builders

```
# paths
ListCronTasksPath / CronActionPath(run|enable|disable, id)

# requests
BuildListCronTasksRequest(base, token) -> GET + optional Bearer
BuildCronActionRequest(base, token, action, id) -> POST + optional Bearer
```

## Preconditions

`Op=cronapi` dispatches to `macosapp/cronapi`. Package mirrors `serviceapi`.

## Steps

1. Leaf sets `CronAPILeaf` and any BaseURL/Token/TaskID/CronAction.

## Context

REQUIREMENT cronapi: list/run/enable/disable paths; auth header when token set.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "cronapi"
	return nil
}
```
