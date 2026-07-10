# Scenario

**Feature**: when task off, Enable is active and Disable is inactive

```
enabled=false -> EnableActive=true, DisableActive=false
```

## Preconditions

User has not enabled periodic backup (default or after Disable).

## Steps

1. Set `Op=menu_gating`, Enabled=false.

## Context

REQUIREMENT #22.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "menu_gating"
	req.Enabled = false
	return nil
}
```
