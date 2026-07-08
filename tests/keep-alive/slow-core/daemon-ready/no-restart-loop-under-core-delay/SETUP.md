# Scenario

**Bug**: prolonged core bind delay must not trigger repeated daemon kill/restart

```
# observe 25s with 15s core sleep — no "failed to become ready" churn
keep-alive(60s) -> stable managed PID -> no ERROR restart loop
```

## Preconditions

`CoreDelayMs=15000`, `StartupTimeout=60s`, extension skipped.

## Steps

1. `ObserveSecs=25` — long enough to catch flapping after first ready signal.

## Context

Catches restart loops that appear after an initial false-negative or race.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.CoreDelayMs = 15000
	req.StartupTimeout = "60s"
	req.ObserveSecs = 25
	req.SkipExtensionStartup = true
	req.WriteExtensionConfig = false
	return nil
}
```