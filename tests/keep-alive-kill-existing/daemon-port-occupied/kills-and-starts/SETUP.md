# Scenario

**Feature**: daemon port conflict resolved by `--kill-existing`

```
lsof :23312 -> SIGTERM occupier -> new daemon binds -> GET /api/keep-alive/status
```

## Preconditions

Parent `daemon-port-occupied` setup.

## Steps

1. `StartupWaitSecs=18`.

## Context

REQUIREMENT leaf: `daemon-port-occupied/kills-and-starts`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.StartupWaitSecs = 18
	return nil
}
```