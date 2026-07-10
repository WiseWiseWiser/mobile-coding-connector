# Scenario

**Feature**: SSE done frame default message

```
FormatBackupProgressDone("") -> "[done] archive ready"
```

## Preconditions

Empty message uses sealed default `archive ready`.

## Steps

1. Op=format_done; empty Message.

## Context

REQUIREMENT format table done row.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "format_done"
	req.Message = ""
	return nil
}
```
