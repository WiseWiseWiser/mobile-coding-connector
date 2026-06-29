# Scenario

**Bug**: keep-alive kills server when extension work blocks port bind past 10s

```
# 15s extension delay must not prevent daemon server_ready within StartupTimeout
keep-alive WaitForPort(10s) -> /ping succeeds while extension still sleeping
```

## Preconditions

Inherited slow-extension config (15s default extension delay).

## Steps

1. Leaves set `ObserveSecs` for stability window.

## Context

Daemon-focused assertions: `ServerReady`, `PortReadyMs`, `RestartLoopSeen`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.ExtensionDelayMs <= 0 {
		req.ExtensionDelayMs = 15000
	}
	return nil
}
```