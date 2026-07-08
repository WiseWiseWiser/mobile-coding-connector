# Scenario

**Feature**: GET /api/grok/usage returns ready JSON

```
server fetch (mock) -> GET :23712/api/grok/usage -> status ready
```

## Preconditions

`mock-success.sh` path exported as `GROK_SHOW_USAGE_COMMAND` in daemon env.

## Steps

1. `MockScript=mock-success.sh`, `WaitAPIReadySecs=15`.

## Context

REQUIREMENT leaf: `api/get-usage-ready`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.MockScript = "mock-success.sh"
	req.WaitAPIReadySecs = 15
	return nil
}
```