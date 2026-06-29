# Scenario

**Feature**: built-in default server when no flags or config

```
# empty config -> http://localhost:<default> -> ping
local-agent ping -> ai-critic-server (injected default port) -> pong
```

## Preconditions

No `local-agent-config.json` seed; no `--server` or `--port`.

## Steps

1. Start server on ephemeral port.
2. `SyncDefaultPortFromServer` injects that port as the built-in default in the child.
3. Run `ping`.

## Context

Tests default resolution without hard-coding port 23712 in the harness.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.StartServer = true
	req.SyncDefaultPortFromServer = true
	req.Args = []string{"ping"}
	return nil
}
```