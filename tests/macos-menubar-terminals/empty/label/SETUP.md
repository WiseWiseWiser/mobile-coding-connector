# Scenario

**Feature**: sealed empty Terminals placeholder text

```
GET /api/terminal/sessions -> [] -> FormatTerminalsEmptyLabel -> "No terminal sessions"
```

## Preconditions

Server returned an empty sessions array for the active endpoint.

## Steps

1. Invoke empty-label formatter via `Op=empty`.

## Context

REQUIREMENT leaf: `empty/label`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "empty"
	return nil
}
```
