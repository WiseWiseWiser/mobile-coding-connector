# Scenario

**Bug**: prolonged extension work must not trigger repeated daemon kill/restart

```
# observe 20s with 15s extension sleep — no "failed to become ready" churn
keep-alive -> stable managed PID -> no ERROR restart loop
```

## Preconditions

15s extension delay, extension config armed.

## Steps

1. `ObserveSecs=20`.

## Context

Catches flapping if port becomes ready then daemon still restarts.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ExtensionDelayMs = 15000
	req.ObserveSecs = 20
	return nil
}
```