# Scenario

**Feature**: remote project list auth failures explain CLI authorization

```
# project list reaches server with bad token
remote-agent project list -> ai-critic-server -> unauthorized

# remote profile guidance points to remote-agent config only
remote-agent stderr -> remote-agent config hint
```

## Preconditions

The server is reachable and initialized; auth rejection is caused by an invalid token,
not a networking failure.

## Steps

1. Child leaves choose the token and subcommand.

## Context

Split factor: remote profile auth failure messaging for `project list`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if len(req.Args) == 0 {
		req.Args = []string{"project", "list"}
	}
	return nil
}
```
