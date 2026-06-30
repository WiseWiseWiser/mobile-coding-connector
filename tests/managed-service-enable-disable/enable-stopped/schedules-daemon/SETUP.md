# Scenario

**Feature**: enable on stopped service defers start to daemon reconcile

```
# immediate snapshot after enable should still be stopped
POST /api/services/enable -> message mentions daemon check

# after one reconcile window the service is running
reconcileProcesses (~5s) -> pid > 0
```

## Preconditions

Parent `enable-stopped` setup disables auto-management and leaves service stopped.

## Steps

1. Inherit parent setup with 7s post-enable wait.
2. Assert deferred-start message, `enabled=true` on disk, and running after wait.

## Context

REQUIREMENT leaf: `enable-stopped/schedules-daemon`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.WaitAfterSecs < 7 {
		req.WaitAfterSecs = 7
	}
	return nil
}
```