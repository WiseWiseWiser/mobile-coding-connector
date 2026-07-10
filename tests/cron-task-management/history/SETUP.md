# Scenario

**Feature**: 7-day cron run history API

```
# runs append CronTaskRun; GET /api/cron-tasks/history?id= returns last 7d (UTC)
manager -> history store -> API/CLI history
```

## Preconditions

1. History is returned in UTC RFC3339 timestamps.
2. Prune older than 7 days on write/tick (or on history read — implementation choice).

## Steps

1. Leaf seeds or forces runs, then reads history.
2. Assert checks presence of recent runs and absence of >7d-old rows.

## Context

Priority leaf 10 (history 7d).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.UseCLI = false
	return nil
}
```
