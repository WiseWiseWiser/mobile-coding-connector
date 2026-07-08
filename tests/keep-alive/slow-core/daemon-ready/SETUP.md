# Scenario

**Bug**: daemon startup timeout must accommodate slow core bind without restart loop

```
# 15s pre-listen delay vs configurable StartupTimeout
keep-alive WaitForPort(timeout) -> TCP listen on 127.0.0.1:P -> server_ready
```

## Preconditions

Inherited slow-core config: `CoreDelayMs=15000`, extension skipped.

## Steps

1. Leaves set `StartupTimeout` (60s success path or 10s negative control).
2. Leaves set `ObserveSecs` for the stability window.

## Context

Daemon-focused assertions: `ServerReady`, `PortReadyMs`, `RestartLoopSeen`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.CoreDelayMs <= 0 {
		req.CoreDelayMs = 15000
	}
	req.SkipExtensionStartup = true
	req.WriteExtensionConfig = false
	return nil
}
```