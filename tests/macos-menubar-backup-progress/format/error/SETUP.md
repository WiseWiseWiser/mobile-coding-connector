# Scenario

**Feature**: SSE error frame line

```
FormatBackupProgressError("stream failed") -> "ERROR: stream failed"
```

## Preconditions

SSE `type=error` or local failure message.

## Steps

1. Op=format_error; Message set.

## Context

REQUIREMENT #11.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "format_error"
	req.Message = "stream failed"
	return nil
}
```
