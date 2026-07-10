# Scenario

**Feature**: list cron tasks path

```
ListCronTasksPath() -> "/api/cron-tasks"
```

## Preconditions

List uses GET `/api/cron-tasks` (no `all=1` query — unlike services).

## Steps

1. Set `CronAPILeaf=list-path`.

## Context

REQUIREMENT leaf: `cronapi/list-path`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.CronAPILeaf = "list-path"
	return nil
}
```
