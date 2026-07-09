# Scenario

**Feature**: exited session with empty name uses id and appends ` [EXITED]`

```
FormatTerminalTitle("","sess-1","exited") -> "sess-1 [EXITED]"
```

## Preconditions

Session name is empty; id is the base; status `exited`.

## Steps

1. Set name `""`, session id `sess-1`, status `exited`.

## Context

REQUIREMENT leaf: `title/exited-empty-name`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Name = ""
	req.SessionID = "sess-1"
	req.Status = "exited"
	return nil
}
```
