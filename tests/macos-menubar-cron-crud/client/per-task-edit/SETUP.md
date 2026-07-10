# Scenario

**Feature**: per-task nested menu includes Edit…

```
ForEach task -> Menu { …; Edit… }  // opens Cron Editor for update
```

## Preconditions

Edit is per-task only (not a top-level Cron action).

## Steps

1. Set `ClientLeaf=per-task-edit`.

## Context

REQUIREMENT leaf: `client/per-task-edit` (scenario 5).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "per-task-edit"
	return nil
}
```
