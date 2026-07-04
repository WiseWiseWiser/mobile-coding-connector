# Scenario

**Feature**: kill server-port occupier before start

```
port occupier (server, /ping) -> keep-alive --kill-existing -> occupier dead -> daemon up
```

## Preconditions

`KillExisting=true`, server port occupier with ping handler.

## Steps

1. Do not occupy daemon port.

## Context

Simulates stale managed server on `--port`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.KillExisting = true
	req.OccupyServerPort = true
	req.OccupyDaemonPort = false
	req.ExpectStart = true
	req.ExpectError = false
	return nil
}
```