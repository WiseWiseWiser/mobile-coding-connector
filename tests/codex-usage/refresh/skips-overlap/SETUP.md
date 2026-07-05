# Scenario

**Feature**: concurrent refresh does not double-fetch

```
two concurrent refresh -> injectable fetch invocation count == 1
```

## Preconditions

Slow injectable fetcher increments an atomic counter and sleeps 2s.

## Steps

1. `FetchMode=slow` (handled by refresh `Run` harness).

## Context

REQUIREMENT leaf: `refresh/skips-overlap`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.FetchMode = "slow"
	return nil
}
```