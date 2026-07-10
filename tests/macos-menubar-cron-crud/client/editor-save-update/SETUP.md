# Scenario

**Feature**: Cron Editor Save wires update (PUT) for existing tasks

```
CronEditorView (edit with id) Save -> updateCronTask / PUT /api/cron-tasks
  -> refresh -> close
```

## Preconditions

Shared or app Swift includes Cron Editor and update client method.

## Steps

1. Set `ClientLeaf=editor-save-update`.

## Context

REQUIREMENT leaf: `client/editor-save-update` (scenario 7).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "editor-save-update"
	return nil
}
```
