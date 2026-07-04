# Scenario

**Feature**: error status shows error message

```
FormatGrokLabel("error","","timeout waiting") -> "Grok timeout waiting for usage"
```

## Preconditions

Daemon surfaces fetch timeout in `error` field.

## Steps

1. Use requirement table message text.

## Context

REQUIREMENT leaf: `label/error`. Matches API error string for usage fetch timeout.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Status = "error"
	req.WeeklyLimit = ""
	req.ErrorMsg = "timeout waiting for usage"
	return nil
}
```