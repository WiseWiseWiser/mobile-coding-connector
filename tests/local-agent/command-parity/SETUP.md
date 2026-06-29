# Scenario

**Feature**: shared subcommands behave like remote-agent against local server

```
# same request/help surfaces as remote-agent profile with local branding
local-agent <subcommand> -> ai-critic-server API
```

## Preconditions

Server optional depending on leaf.

## Steps

1. Configure subcommand per leaf.

## Context

Command parity requirement with local-specific help strings.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.Args == nil {
		req.Args = []string{}
	}
	return nil
}
```