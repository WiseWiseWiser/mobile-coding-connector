# Scenario

**Feature**: success footer line

```
FormatBackupProgressStatusSuccess() -> "Status: Success"
```

## Preconditions

Terminal success after write.

## Steps

1. Op=format_status_success.

## Context

REQUIREMENT #13.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "format_status_success"
	return nil
}
```
