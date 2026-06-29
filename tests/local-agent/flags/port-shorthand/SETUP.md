# Scenario

**Feature**: `--port` resolves to localhost URL

```
# --port N -> http://localhost:N -> GET /ping on running server
local-agent --port N -> ai-critic-server -> pong
```

## Preconditions

Server listens on the same port passed via `--port`.

## Steps

1. `StartServer = true`, `SyncPortFlagFromServer = true` so `--port` matches the bound server.
2. Run `ping` with no `--server`.

## Context

REQUIREMENT: `--port` is shorthand for `--server http://localhost:N`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.StartServer = true
	req.SyncPortFlagFromServer = true
	req.Args = []string{"ping"}
	return nil
}
```