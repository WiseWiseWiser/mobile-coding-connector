# Scenario

**Feature**: error service title

```
FormatServiceTitle(...,"error",...) -> "… ⚠ Error"
```

## Preconditions

Service status is `error`; title uses error presentation per menubar spec.

## Steps

1. Set status `error` (name may be truncated in output).

## Context

REQUIREMENT leaf: `title/error`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Name = "web"
	req.Status = "error"
	req.Enabled = true
	return nil
}
```