# Scenario

**Feature**: non-empty session name is the title when status is running

```
FormatTerminalTitle("demo","abc","running") -> "demo"
```

## Preconditions

Session has a non-empty display name distinct from id; status is running (no suffix).

## Steps

1. Set name `demo`, session id `abc`, status `running`.

## Context

REQUIREMENT leaf: `title/with-name` (running → base only).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Name = "demo"
	req.SessionID = "abc"
	req.Status = "running"
	return nil
}
```
