# Scenario

**Feature**: empty session name falls back to id; empty status adds no suffix

```
FormatTerminalTitle("","sess-1","") -> "sess-1"
```

## Preconditions

Session name is the empty string; id is stable; status empty (same as running for title).

## Steps

1. Set name `""`, session id `sess-1`, status `""`.

## Context

REQUIREMENT leaf: `title/empty-name` (empty/unknown status → no suffix).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Name = ""
	req.SessionID = "sess-1"
	req.Status = ""
	return nil
}
```
