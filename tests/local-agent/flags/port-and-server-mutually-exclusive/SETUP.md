# Scenario

**Feature**: `--port` and `--server` cannot be used together

```
# both globals set -> usage error, no dial
local-agent --port N --server URL -> (reject before network)
```

## Preconditions

No server required; reachability must not run before flag validation.

## Steps

1. Set `Port` to 8080 and `Server` to `http://example.com`.
2. Subcommand `ping` (any command suffices once globals are invalid).

## Context

REQUIREMENT: mutually exclusive globals with a clear usage message.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Port = 8080
	req.Server = "http://example.com"
	req.Args = []string{"ping"}
	req.StartServer = false
	return nil
}
```