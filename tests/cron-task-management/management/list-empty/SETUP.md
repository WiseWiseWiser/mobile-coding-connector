# Scenario

**Feature**: list cron tasks when none are defined

```
# fresh AI_CRITIC_HOME without cron-tasks.json (or empty)
GET /api/cron-tasks -> []
```

## Preconditions

1. No seed tasks; config home has no prior cron definitions.

## Steps

1. Start server with empty cron state.
2. Action `list` (default) → snapshot `Tasks`.

## Context

CRUD priority: empty list baseline.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedTasks = nil
	req.Action = "list"
	return nil
}
```
