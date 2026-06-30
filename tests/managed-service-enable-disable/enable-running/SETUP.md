# Scenario

**Feature**: enable a service that is already running

```
# disabled definition but manually started process
services.json(enabled=false) -> POST /api/services/start -> running

# enable must not disrupt the running process
POST /api/services/enable -> already running prompt
```

## Preconditions

1. Target service is disabled in `services.json`.
2. Service is manually started before enable (Start works regardless of `enabled`).

## Steps

1. Seed disabled `sleep` service.
2. Pre-start via `Request.PreStartID`.
3. Call `POST /api/services/enable` without extra wait.

## Context

Sibling `enable-stopped/` covers deferred daemon start.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Services = []ServiceSeed{
		sleepService("svc-en-run-001", "sleep-manual", boolPtr(false)),
	}
	req.TargetID = "svc-en-run-001"
	req.Action = "enable"
	req.PreStartID = "svc-en-run-001"
	req.WaitAfterSecs = 0
	return nil
}
```