# Scenario

**Bug**: daemon must mark server ready within 10s despite 15s extension delay

```
keep-alive -> WaitForPort -> server_ready waited_ms<10000
```

## Preconditions

`ExtensionDelayMs=15000`, extension config armed.

## Steps

1. `ObserveSecs=12` to capture first ready signal.

## Context

Primary regression for production restart loop.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ExtensionDelayMs = 15000
	req.ObserveSecs = 12
	return nil
}
```