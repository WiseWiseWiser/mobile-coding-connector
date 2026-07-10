# Scenario

**Feature**: Cron Editor Save wires create (POST) for new tasks

```
CronEditorView (isNew / no id) Save -> createCronTask / POST /api/cron-tasks
  -> refresh -> close
```

## Preconditions

Shared or app Swift includes Cron Editor and create client method.

## Steps

1. Set `ClientLeaf=editor-save-create`.

## Context

REQUIREMENT leaf: `client/editor-save-create` (scenario 7).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "editor-save-create"
	return nil
}
```
