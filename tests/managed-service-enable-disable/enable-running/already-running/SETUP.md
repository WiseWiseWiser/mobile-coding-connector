# Scenario

**Feature**: enable on running service returns already-running prompt

```
# manually started disabled service still running after enable
POST /api/services/enable?id=svc-en-run-001 -> already running message, pid unchanged
```

## Preconditions

Parent `enable-running` has started the service and called enable.

## Steps

1. Inherit parent setup.
2. Assert prompt, `enabled=true` on disk, and unchanged live PID.

## Context

REQUIREMENT leaf: `enable-running/already-running`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.WaitAfterSecs = 0
	if req.PreStartID == "" {
		req.PreStartID = req.TargetID
	}
	return nil
}
```