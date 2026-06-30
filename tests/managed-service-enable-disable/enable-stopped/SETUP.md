# Scenario

**Feature**: enable a stopped disabled service schedules daemon start

```
# disabled service stopped at enable time
services.json(enabled=false) -> server boot -> stopped

# enable sets desired=true without synchronous start
POST /api/services/enable -> daemon starts within ~6s
```

## Preconditions

1. Target service is disabled (`enabled: false`) and not running.
2. Post-enable wait covers one reconcile ticker window (default 7s).

## Steps

1. Seed disabled `sleep` service.
2. Call `POST /api/services/enable`.
3. Wait `Request.WaitAfterSecs` (7s) for daemon reconcile.

## Context

Sibling `enable-running/` covers enabling while already running.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Services = []ServiceSeed{
		sleepService("svc-en-stopped-001", "sleep-disabled", boolPtr(false)),
	}
	req.TargetID = "svc-en-stopped-001"
	req.Action = "enable"
	req.WaitAfterSecs = 7
	return nil
}
```