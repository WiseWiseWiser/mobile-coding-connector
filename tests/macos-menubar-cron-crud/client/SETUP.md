# Scenario

**Feature**: Swift source contracts for Cron CRUD menus and Cron Editor

```
# both apps
Menu("Cron")
  tasks ▸ { …; Edit…; Delete… (disabled if running) }
  ────────
  New Cron Task…   // bottom; disabled if remote not configured

# Cron Editor
Save -> createCronTask (POST) | updateCronTask (PUT) -> refresh -> close
models: CronTaskDefinition / status fields for prefill
```

## Preconditions

`Op=client` reads Swift under local, remote, and Shared. No UI automation.

## Steps

1. Leaf sets `ClientLeaf`.
2. Root inspects sources and sets contract flags.

## Context

REQUIREMENT scenarios 4–7: New Cron Task…, per-task Edit/Delete, delete
disabled when running, editor Save create/update.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "client"
	return nil
}
```
