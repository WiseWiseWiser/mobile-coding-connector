# Scenario

**Feature**: start hint when local server is down

```
# resolved URL not listening -> stderr mentions ai-critic
local-agent ping -> reachability false -> Error + Start the server with: ai-critic
```

## Preconditions

Reachability mocked to false; built-in default port injected so resolution is deterministic.

## Steps

1. Do not start server.
2. `InjectedDefaultPort = 23712` (or any fixed port) with `MockReachability = false`.
3. Run `ping` without flags.

## Context

REQUIREMENT: non-listening server shows `ai-critic` start suggestion.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	down := false
	req.MockReachability = &down
	req.InjectedDefaultPort = 23712
	req.Args = []string{"ping"}
	req.StartServer = false
	return nil
}
```