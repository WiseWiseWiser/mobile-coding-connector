# Scenario

**Feature**: reachability check and start hints

```
# resolve URL -> reachability -> (hint | normal API error)
local-agent -> reachability gate -> command or stderr hint
```

## Preconditions

Leaves control mock reachability and whether the server process runs.

## Steps

1. Configure `MockReachability`, `StartServer`, and auth flags per leaf.

## Context

Start hint must appear only when the server is not listening, not on auth failure.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.Args == nil {
		req.Args = []string{}
	}
	return nil
}
```