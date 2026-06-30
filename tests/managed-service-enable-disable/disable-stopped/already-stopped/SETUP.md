# Scenario

**Feature**: disable stopped service returns already-stopped prompt

```
# stopped enabled service
POST /api/services/disable?id=svc-stop-001 -> already stopped message
```

## Preconditions

Parent `disable-stopped` setup leaves the service stopped.

## Steps

1. Inherit parent setup.
2. Assert message and `enabled=false` persistence.

## Context

REQUIREMENT leaf: `disable-stopped/already-stopped`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.WaitAfterSecs = 0
	req.PreStartID = ""
	return nil
}
```