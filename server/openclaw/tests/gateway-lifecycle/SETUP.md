# Scenario

**Feature**: mock gateway start, stop, status, and dry-run

```
# Start writes state.json and generated openclaw.json
Manager.Start -> state (running, mock_pid=4242) + WriteGeneratedConfig

# second start rejected; stop is idempotent
Start (running) -> Start -> ALREADY_RUNNING (409)
Stop (stopped) -> OK (no-op)
```

## Preconditions

Valid or invalid config per leaf; optional pre-started gateway.

## Steps

1. Leaf sets lifecycle operation and pre-state.
2. `Run` invokes Manager or API and collects state/status.

## Context

Covers mock PID, conflict mapping, idempotent stop, status slack fields, dry-run.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.GatewayPort == 0 {
		req.GatewayPort = 18789
	}
	return nil
}
```