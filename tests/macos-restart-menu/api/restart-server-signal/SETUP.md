# Scenario

**Feature**: POST `/api/keep-alive/restart` signals managed server restart only

```
# daemon PID unchanged; server child respawns with new PID
POST /api/keep-alive/restart -> status restart_requested -> server PID changes
```

## Preconditions

Daemon running with managed server on test port.

## Steps

1. Set `Op=api-restart-server`.
2. `SettleWaitSecs=20` for server respawn.

## Context

Documents legacy signal path the macOS menu must **stop** using.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "api-restart-server"
	req.SettleWaitSecs = 20
	return nil
}
```