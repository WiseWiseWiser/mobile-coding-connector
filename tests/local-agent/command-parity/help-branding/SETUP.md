# Scenario

**Feature**: top-level help documents local-agent profile

```
# -h prints usage with local-agent name, --port, and default port 23712
local-agent -h -> stdout help text
```

## Preconditions

No server required.

## Steps

1. `GlobalHelp = true`.

## Context

REQUIREMENT: help mentions `local-agent`, `--port`, and built-in default 23712.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.GlobalHelp = true
	req.StartServer = false
	return nil
}
```