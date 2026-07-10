# Scenario

**Feature**: delete cron task removes it from list

```
# seed one task, DELETE /api/cron-tasks?id=
delete -> list no longer contains id
```

## Preconditions

1. Seed a single interval task with known id `to-remove`.

## Steps

1. Seed task before server start.
2. Action `delete` with `Target=to-remove`.
3. Assert list does not contain the id.

## Context

CRUD priority: remove.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedTasks = []TaskSeed{
		{
			ID:           "to-remove",
			Name:         "to-remove",
			Command:      "echo gone",
			ScheduleMode: "interval",
			Interval:     "1h",
			Enabled:      boolPtr(false),
		},
	}
	req.Action = "delete"
	req.Target = "to-remove"
	return nil
}
```
