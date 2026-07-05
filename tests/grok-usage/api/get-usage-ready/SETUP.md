# Scenario

**Feature**: GET /api/grok/usage returns ready JSON

```
daemon in-process fetch (GROK_SHOW_USAGE_COMMAND) -> GET /api/grok/usage -> status ready
```

## Preconditions

Fake Grok TUI via `GROK_SHOW_USAGE_COMMAND`.

## Steps

1. `WaitAPIReadySecs=15`.
2. Default fake TUI from root `Run` when `ShowUsageCommand` empty.

## Context

REQUIREMENT leaf: `api/get-usage-ready`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.WaitAPIReadySecs = 15
	return nil
}
```