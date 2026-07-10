# Scenario

**Feature**: failure footer line

```
FormatBackupProgressStatusFailed() -> "Status: Failed"
```

## Preconditions

Terminal failure after ERROR line (or guard).

## Steps

1. Op=format_status_failed.

## Context

REQUIREMENT #11 failure end.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "format_status_failed"
	return nil
}
```
