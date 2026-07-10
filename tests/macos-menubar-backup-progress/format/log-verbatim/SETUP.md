# Scenario

**Feature**: SSE log frames print message only (verbatim style)

```
FormatBackupProgressLog("dry-run: machine backup plan") -> "dry-run: machine backup plan"
```

## Preconditions

Sealed style: **no** `[log]` prefix (verbatim message body).

## Steps

1. Op=format_log; Message set.

## Context

REQUIREMENT log row — pick verbatim sealed style.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "format_log"
	req.Message = "dry-run: machine backup plan"
	return nil
}
```
