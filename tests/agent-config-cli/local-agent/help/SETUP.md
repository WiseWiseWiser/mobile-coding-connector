# Scenario

**Feature**: local-agent config help / bare

```
# bare local-agent config prints local branding help
local-agent config -> stdout help (local-agent)
```

## Preconditions

No seed required.

## Steps

1. Child sets bare args.

## Context

local-agent parity for bare help.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedConfig = nil
	return nil
}
```
