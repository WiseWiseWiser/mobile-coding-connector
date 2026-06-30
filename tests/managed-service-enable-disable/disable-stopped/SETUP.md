# Scenario

**Feature**: disable a stopped managed service

```
# enabled service seeded but never started
services.json -> server boot -> status stopped

# disable on stopped service
POST /api/services/disable -> enabled=false, already-stopped prompt
```

## Preconditions

1. Target service is enabled before disable.
2. Service has never been started (or is stopped) at action time.

## Steps

1. Seed one enabled `sleep` service without pre-start.
2. Call `POST /api/services/disable`.

## Context

Sibling `disable-running/` covers the running variant.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Services = []ServiceSeed{
		sleepService("svc-stop-001", "sleep-stopped", boolPtr(true)),
	}
	req.TargetID = "svc-stop-001"
	req.Action = "disable"
	return nil
}
```