# Scenario

**Bug**: macOS menu still restarts managed server via signal path

```
# current (wrong): Restart Server -> restartServer() -> POST /api/keep-alive/restart
# desired: Restart Daemon -> restartDaemon() -> POST /api/keep-alive/restart-daemon
AICriticApp.swift -> DaemonClient.swift -> contract map
```

## Preconditions

Contract expectations are fixed in root `DOCTEST.md` constants.

## Steps

1. `Run` reads `AICriticApp.swift` and `DaemonClient.swift`.
2. Extract menu label, handler method, and restart URL path.

## Context

REQUIREMENT leaf: `client/macos-menu-contract`. RED until Swift implementer updates
menu label, `DaemonClient.restartDaemon()`, and wires the button.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "client-restart"
	return nil
}
```