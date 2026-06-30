# Scenario

**Feature**: disable an already-running managed service

```
# enabled service started before disable
POST /api/services/start -> running sleep process

# disable must not stop the process immediately
POST /api/services/disable -> enabled=false, PID still alive
```

## Preconditions

1. Target service is enabled (default or explicit `enabled: true`).
2. Service process is running before the disable action.

## Steps

1. Seed one enabled `sleep` service.
2. Pre-start the service via API (`Request.PreStartID`).
3. Call `POST /api/services/disable` for the target id.

## Context

Sibling `disable-stopped/` covers the stopped variant with the
already-stopped prompt.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Services = []ServiceSeed{
		sleepService("svc-run-001", "sleep-running", boolPtr(true)),
	}
	req.TargetID = "svc-run-001"
	req.Action = "disable"
	req.PreStartID = "svc-run-001"
	return nil
}
```