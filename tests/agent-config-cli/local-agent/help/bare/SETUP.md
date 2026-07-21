# Scenario

**Feature**: bare local-agent config prints help (does not start UI)

```
# bare config -> help with local-agent name; no Config UI
local-agent config -> stdout help
```

## Preconditions

Empty HOME.

## Steps

1. Args = `["config"]`.

## Context

local-agent parity T1.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Args = []string{"config"}
	return nil
}
```
