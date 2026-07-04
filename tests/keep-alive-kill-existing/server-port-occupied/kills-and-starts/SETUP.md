# Scenario

**Feature**: server port conflict resolved by `--kill-existing`

```
lsof server:PORT -> SIGTERM occupier -> daemon starts server on PORT
```

## Preconditions

Parent `server-port-occupied` setup.

## Steps

1. `StartupWaitSecs=18`.

## Context

REQUIREMENT leaf: `server-port-occupied/kills-and-starts`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.StartupWaitSecs = 18
	return nil
}
```