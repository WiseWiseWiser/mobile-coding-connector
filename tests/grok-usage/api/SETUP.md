# Scenario

**Feature**: HTTP API for grok usage on main server port

```
keep-alive spawns server + GROK_SHOW_USAGE_COMMAND -> GET :23712/api/grok/usage
```

## Preconditions

Server exposes grok usage route on port `23712`; `GROK_SHOW_USAGE_COMMAND` in env; session lock held.

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