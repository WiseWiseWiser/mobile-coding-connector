# Scenario

**Feature**: local project list auth failures explain CLI authorization

```
# server listening -> bad token -> project list auth error with local guidance
local-agent project list -> ai-critic-server -> unauthorized + local-agent config hint

# local profile may guide users toward local credential files
local-agent stderr -> ~/.ai-critic/server-credentials hint
```

## Preconditions

Server is reachable and initialized with the test credential; the CLI passes an explicit bad token.

## Steps

1. Start server on an ephemeral port and sync `--server`.
2. Set `--token definitely-wrong-token`.
3. Run `project list`.

## Context

This extends the auth-failure branch from `auth status` to a user-facing command that
performs an API call and should print actionable local-agent authorization guidance.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	up := true
	req.MockReachability = &up
	req.StartServer = true
	req.SyncServerFromBoundPort = true
	req.TokenSpecified = true
	req.Token = "definitely-wrong-token"
	req.Args = []string{"project", "list"}
	return nil
}
```
