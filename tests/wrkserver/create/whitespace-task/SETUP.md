# Scenario

**Feature**: whitespace-only task treated as no task

```
# task "   " (spaces) -> no slug (UI-friendly empty)
CreateWorktree(..., task="   ") -> same naming as omit task
```

## Preconditions

Clean main git repo.

## Steps

1. Create clean main as `ProjectPath`.
2. Set `Task` to three spaces.
3. POST create (task field present but whitespace-only).

## Context

REQUIREMENT scenario 7. HTTP API treats whitespace as empty → no slug
(differs from CLI which may reject empty task text).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ProjectPath = mkCleanMainRepo(t)
	req.Task = "   "
	req.OmitTask = false
	return nil
}
```
