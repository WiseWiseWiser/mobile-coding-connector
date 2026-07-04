# Scenario

**Feature**: GET /api/grok/usage returns ready JSON

```
daemon fetch (mock) -> GET /api/grok/usage -> status ready
```

## Preconditions

`mock-success.sh` as `GROK_SHOW_USAGE_BIN`.

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