# Scenario

**Bug**: documents old broken behavior — 10s timeout kills server still starting core bind

```
# 15s pre-listen delay with legacy 10s StartupTimeout → kill/restart loop
keep-alive(10s) -> timeout -> failed to become ready -> respawn
```

## Preconditions

`CoreDelayMs=15000`, `StartupTimeout=10s`, extension skipped.

## Steps

1. `ObserveSecs=15` — long enough for at least one startup-timeout failure cycle.

## Context

Negative control: must fail until fix lands; after fix this leaf still documents
that an explicit 10s timeout cannot survive a 15s core delay.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.CoreDelayMs = 15000
	req.StartupTimeout = "10s"
	req.ObserveSecs = 15
	req.SkipExtensionStartup = true
	req.WriteExtensionConfig = false
	return nil
}
```