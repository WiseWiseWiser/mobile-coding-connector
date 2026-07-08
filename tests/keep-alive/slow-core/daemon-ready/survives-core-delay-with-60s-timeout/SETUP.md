# Scenario

**Bug**: 15s core bind delay must succeed when daemon allows 60s startup timeout

```
# hook delays net.Listen 15s; daemon --startup-timeout 60s waits it out
keep-alive(60s) -> managed server (15s pre-listen) -> server_ready waited_ms∈[15000,60000]
```

## Preconditions

`CoreDelayMs=15000`, `StartupTimeout=60s`, extension skipped.

## Steps

1. `ObserveSecs=65` to capture ready after core delay.

## Context

Primary fix validation for remote kill/restart loop under slow fork/I/O.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.CoreDelayMs = 15000
	req.StartupTimeout = "60s"
	req.ObserveSecs = 65
	req.SkipExtensionStartup = true
	req.WriteExtensionConfig = false
	return nil
}
```