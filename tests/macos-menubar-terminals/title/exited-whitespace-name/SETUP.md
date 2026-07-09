# Scenario

**Feature**: exited session with whitespace-only name uses id and appends ` [EXITED]`

```
FormatTerminalTitle("  \t  ","sess-1","exited") -> "sess-1 [EXITED]"
```

## Preconditions

Name is whitespace-only (treated as empty); status `exited`.

## Steps

1. Set name to spaces + tab, session id `sess-1`, status `exited`.

## Context

REQUIREMENT leaf: whitespace name + exited → `sess-1 [EXITED]`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Name = "  \t  "
	req.SessionID = "sess-1"
	req.Status = "exited"
	return nil
}
```
