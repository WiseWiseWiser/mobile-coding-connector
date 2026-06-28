# Scenario

**Feature**: GET /api/openclaw/doctor health report

```
# Doctor aggregates server checks from config and runtime state
Manager.Doctor -> checks[] (mock_mode, node, slack, gateway, generated_config)

# mocked integration always warns; real deps checked on PATH
mock_mode -> warn; node/openclaw_cli -> ok|fail with hint
```

## Preconditions

Config and optional running state per leaf.

## Steps

1. Leaf seeds config/state (optional `PreStart`).
2. `Run` calls `Manager.Doctor()`.

## Context

Fills gap left by unit tests: full doctor check matrix and hints.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.Op == "" {
		req.Op = OpDoctor
	}
	if req.GatewayPort == 0 {
		req.GatewayPort = 18789
	}
	return nil
}
```