# Scenario

**Feature**: non-empty session name is the title

```
FormatTerminalTitle("demo","abc") -> "demo"
```

## Preconditions

Session has a non-empty display name distinct from id.

## Steps

1. Set name `demo`, session id `abc`.

## Context

REQUIREMENT leaf: `title/with-name`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Name = "demo"
	req.SessionID = "abc"
	return nil
}
```
