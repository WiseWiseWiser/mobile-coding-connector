# Scenario

**Feature**: auth errors must not show the start hint

```
# server listening -> reachability ok -> bad token -> auth error without ai-critic hint
local-agent auth status -> server up -> unauthorized (no start hint)
```

## Preconditions

Server running; reachability passes (real or mocked up); explicit bad token.

## Steps

1. Start server on ephemeral port.
2. `SyncServerFromBoundPort = true`.
3. `TokenSpecified = true`, `Token = "definitely-wrong-token"`.
4. Run `auth status`.

## Context

REQUIREMENT: distinguish unreachable server from auth failure.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	up := true
	req.MockReachability = &up
	req.StartServer = true
	req.SyncServerFromBoundPort = true
	req.TokenSpecified = true
	req.Token = "definitely-wrong-token"
	req.Args = []string{"auth", "status"}
	return nil
}
```