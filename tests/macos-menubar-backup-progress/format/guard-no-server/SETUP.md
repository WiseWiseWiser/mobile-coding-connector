# Scenario

**Feature**: guard failure — no server selected (visible in window)

```
FormatBackupProgressGuardError("no_server") -> "ERROR: no server selected"
```

## Preconditions

Empty server name early failure surfaces in progress lines.

## Steps

1. Op=format_guard; GuardReason=no_server.

## Context

REQUIREMENT goal 5; guard table.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "format_guard"
	req.GuardReason = "no_server"
	return nil
}
```
