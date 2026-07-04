# Scenario

**Feature**: `--kill-existing` with no port conflicts

```
free ports -> keep-alive --kill-existing -> daemon status running
```

## Preconditions

No occupiers; `KillExisting=true`.

## Steps

1. Set `ExpectStart=true`, occupier flags false.

## Context

Happy path for macOS menu-bar spawn semantics.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.KillExisting = true
	req.OccupyServerPort = false
	req.OccupyDaemonPort = false
	req.ExpectStart = true
	req.ExpectError = false
	return nil
}
```