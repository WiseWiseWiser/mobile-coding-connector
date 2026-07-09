# Scenario

**Feature**: whitespace-only session name falls back to id

```
FormatTerminalTitle("  \t  ","sess-1") -> "sess-1"
```

## Preconditions

Name is non-empty only by whitespace; TrimSpace treats it as empty.

## Steps

1. Set name to spaces + tab, session id `sess-1`.

## Context

REQUIREMENT: empty **or whitespace** name → id.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Name = "  \t  "
	req.SessionID = "sess-1"
	return nil
}
```
