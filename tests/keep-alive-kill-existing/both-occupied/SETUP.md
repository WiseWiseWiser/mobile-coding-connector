# Scenario

**Feature**: kill both port occupiers before start

```
occupiers (server + daemon) -> keep-alive --kill-existing -> both dead -> clean start
```

## Preconditions

`KillExisting=true`, both occupier flags set.

## Steps

1. Inherit defaults for expectations.

## Context

Worst-case stale listeners on both ports (menu-bar respawn path).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.KillExisting = true
	req.OccupyServerPort = true
	req.OccupyDaemonPort = true
	req.ExpectStart = true
	req.ExpectError = false
	return nil
}
```