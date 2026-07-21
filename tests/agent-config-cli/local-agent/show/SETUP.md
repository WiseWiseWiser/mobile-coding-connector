# Scenario

**Feature**: local-agent config --show

```
# --show reads local-agent-config.json only
local-agent config --show -> pretty JSON from local path
```

## Preconditions

Optional seed; may place remote sentinel for isolation.

## Steps

1. Child seeds local (and optional remote) config.

## Context

local-agent path isolation.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if len(req.Args) == 0 {
		req.Args = []string{"config", "--show"}
	}
	return nil
}
```
