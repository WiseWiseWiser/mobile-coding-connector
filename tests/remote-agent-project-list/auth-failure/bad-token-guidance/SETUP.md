# Scenario

**Feature**: remote project list bad token prints remote-agent config hint

```
# bad token rejects API call
remote-agent project list -> GET /api/projects?all=true -> unauthorized

# remote-agent should not mention local credential files
unauthorized -> remote-agent config hint
```

## Preconditions

No local-only credential guidance applies to the remote profile.

## Steps

1. Use token `definitely-wrong-token`.
2. Run `project list` against a reachable server.

## Context

This leaf covers the remote-agent half of profile-specific friendly auth failures.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Token = "definitely-wrong-token"
	req.Args = []string{"project", "list"}
	return nil
}
```
