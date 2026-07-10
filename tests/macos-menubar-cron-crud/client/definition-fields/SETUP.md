# Scenario

**Feature**: Swift models expose Cron definition fields for editor prefill

```
CronTaskDefinition | CronTaskStatus (+ command, workingDir, timeout, …)
  -> editor fields: name, command, workingDir, scheduleMode, interval,
     cronExpr, timeout, enabled
```

## Preconditions

Shared models extended beyond operate-only title fields; no extraEnv in UI.

## Steps

1. Set `ClientLeaf=definition-fields`.

## Context

REQUIREMENT: definition body fields; CronTaskDefinition + status for prefill.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "definition-fields"
	return nil
}
```
