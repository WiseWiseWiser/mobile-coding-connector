# Scenario

**Feature**: concurrent refresh does not double-fetch

```
two concurrent refresh -> mock counter == 1
```

## Preconditions

`mock-slow.sh` with `GROK_MOCK_COUNTER_FILE`.

## Steps

1. `MockScript=mock-slow.sh`.

## Context

REQUIREMENT leaf: `refresh/skips-overlap`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.MockScript = "mock-slow.sh"
	return nil
}
```