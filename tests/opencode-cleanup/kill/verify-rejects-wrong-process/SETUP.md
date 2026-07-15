# Scenario

**Feature**: Kill skips non-opencode process even when registry points at it

```
stdlib http listener + registry pid -> Kill -> process still alive
```

## Preconditions

- Registry fixture references PID of plain `net/http` listener, not opencode.

## Steps

1. `Op = OpKill`, `StartWrongProcess = true`, `UseRegistryPID = true`.

## Context

Safety guard: never kill arbitrary PIDs from stale registry.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpKill
	req.StartWrongProcess = true
	req.UseRegistryPID = true
	return nil
}
```
