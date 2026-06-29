# Scenario

**Feature**: default server URL when globals and saved domain are absent

```
# no --server/--port -> built-in default or saved config entry
local-agent -> resolve URL -> ping/auth against server
```

## Preconditions

Leaves differ on whether `local-agent-config.json` is seeded.

## Steps

1. Configure `StartServer`, config seed, and subcommand per leaf.

## Context

Resolution priority: `--server` > `--port` > saved default > built-in `http://localhost:23712` (hooked in tests).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.Args == nil {
		req.Args = []string{}
	}
	return nil
}
```