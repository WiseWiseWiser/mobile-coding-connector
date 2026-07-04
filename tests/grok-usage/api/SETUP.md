# Scenario

**Feature**: HTTP API for grok usage on keep-alive daemon

```
keep-alive + GROK_SHOW_USAGE_BIN -> GET /api/grok/usage
```

## Preconditions

Daemon exposes grok usage route; session lock held.

## Steps

1. Set `Op=api` in leaf.

## Context

End-to-end API contract for Swift menu-bar client.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "api"
	return nil
}
```