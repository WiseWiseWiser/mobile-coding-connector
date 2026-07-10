# Scenario

**Feature**: CanRunBackupNow — Backup Now enablement independent of task enabled

```
# Backup Now is a one-shot; Enable only controls the hourly schedule
hasEndpoint && !running && serverName != "" -> CanRunBackupNow = true
# enabled is NOT an input to the helper
```

## Preconditions

`Op=can_run`. Helper signature does not include `enabled`.

## Steps

1. Leaf sets HasEndpoint, Running, ServerName (and Enabled for story only).

## Context

REQUIREMENT scenarios 1–5 and 14 (one-shot when disabled).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "can_run"
	return nil
}
```
