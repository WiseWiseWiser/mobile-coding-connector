# Scenario

**Feature**: when task on, Disable is active and Enable is inactive

```
enabled=true -> EnableActive=false, DisableActive=true
```

## Preconditions

User enabled periodic backup.

## Steps

1. Set `Op=menu_gating`, Enabled=true.

## Context

REQUIREMENT #23.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "menu_gating"
	req.Enabled = true
	return nil
}
```
