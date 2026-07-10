# Scenario

**Feature**: Backup Now menu item not gated on backupEnabled

```
Button("Backup Now…").disabled(!hasEndpoint || backupRunning)
# must NOT include backupEnabled in the disabled expression
```

## Preconditions

One-shot policy reflected in Swift menu gating (matches CanRunBackupNow).

## Steps

1. ClientLeaf=not-gated-on-enabled.

## Context

REQUIREMENT #18.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "not-gated-on-enabled"
	return nil
}
```
