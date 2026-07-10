# Scenario

**Feature**: cron task definition CRUD (list / create / remove)

```
# empty or created definitions under AI_CRITIC_HOME/cron-tasks.json
HTTP GET|POST|DELETE /api/cron-tasks -> global list of CronTaskStatus
```

## Preconditions

1. Server started with isolated config home (possibly empty of cron tasks).
2. Actions use authenticated HTTP API (not CLI) unless a leaf overrides.

## Steps

1. Leaf sets `Action` to list, create, or delete.
2. Run snapshots `Tasks` after the action.
3. Assert checks empty list, presence after create, or absence after delete.

## Context

Priority leaves 1 (CRUD). Global scope only — no projectDir.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.UseCLI = false
	return nil
}
```
