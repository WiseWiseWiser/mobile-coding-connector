# Scenario

**Feature**: POST `/api/keep-alive/restart-daemon` exec-replaces daemon via SSE

```
# macOS client will POST + drain body (no log parsing); test verifies SSE done + recovery
POST /api/keep-alive/restart-daemon -> SSE done.success=true -> status + /ping back
```

## Preconditions

Daemon running; management API on port `23312`.

## Steps

1. Set `Op=api-restart-daemon`.
2. `SettleWaitSecs=25` for exec replace and server respawn.

## Context

Matches web `restartDaemonStreaming()` and future Swift `restartDaemon()` drain behavior.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "api-restart-daemon"
	req.SettleWaitSecs = 25
	return nil
}
```