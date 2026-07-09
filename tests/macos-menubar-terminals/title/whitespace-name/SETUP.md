# Scenario

**Feature**: whitespace-only session name falls back to id when status is running

```
FormatTerminalTitle("  \t  ","sess-1","running") -> "sess-1"
```

## Preconditions

Name is non-empty only by whitespace; TrimSpace treats it as empty; status running.

## Steps

1. Set name to spaces + tab, session id `sess-1`, status `running`.

## Context

REQUIREMENT: empty **or whitespace** name → id; running → no `[EXITED]` suffix.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Name = "  \t  "
	req.SessionID = "sess-1"
	req.Status = "running"
	return nil
}
```
