# Scenario

**Feature**: empty session name falls back to id

```
FormatTerminalTitle("","sess-1") -> "sess-1"
```

## Preconditions

Session name is the empty string; id is stable.

## Steps

1. Set name `""`, session id `sess-1`.

## Context

REQUIREMENT leaf: `title/empty-name`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Name = ""
	req.SessionID = "sess-1"
	return nil
}
```
