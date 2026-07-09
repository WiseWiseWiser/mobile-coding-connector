# Scenario

**Feature**: exited session with non-empty name appends ` [EXITED]`

```
FormatTerminalTitle("demo","abc","exited") -> "demo [EXITED]"
```

## Preconditions

Session has a display name and status `exited` (listed in Terminals menu).

## Steps

1. Set name `demo`, session id `abc`, status `exited`.

## Context

REQUIREMENT leaf: `title/exited-with-name`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Name = "demo"
	req.SessionID = "abc"
	req.Status = "exited"
	return nil
}
```
