# Scenario

**Feature**: nested Status menu with Enable and Disable only as children

```
Backup -> Status: … ▸ Enable | Disable
```

## Preconditions

Status title carries state; Enable/Disable are the only nested actions under Status.

## Steps

1. Set `ClientLeaf=status-enable-disable`.

## Context

REQUIREMENT #25.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "status-enable-disable"
	return nil
}
```
