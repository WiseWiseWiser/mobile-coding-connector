# Scenario

**Feature**: legacy port conflict without `--kill-existing`

```
port occupier (server) -> keep-alive (no flag) -> startup error
```

## Preconditions

`KillExisting=false`, server port occupied.

## Steps

1. Set `ExpectError=true`, `ExpectStart=false`.

## Context

Preserves existing behavior when flag omitted.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.KillExisting = false
	req.OccupyServerPort = true
	req.OccupyDaemonPort = false
	req.ExpectStart = false
	req.ExpectError = true
	req.StartupWaitSecs = 8
	return nil
}
```