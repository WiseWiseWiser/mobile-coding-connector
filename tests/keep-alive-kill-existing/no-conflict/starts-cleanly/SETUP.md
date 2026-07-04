# Scenario

**Feature**: clean start with `--kill-existing`

```
keep-alive --kill-existing --port P -> managed server -> GET /api/keep-alive/status running
```

## Preconditions

Parent `no-conflict` setup: free ports, flag set.

## Steps

1. `StartupWaitSecs=18`.

## Context

REQUIREMENT leaf: `no-conflict/starts-cleanly`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.StartupWaitSecs = 18
	return nil
}
```