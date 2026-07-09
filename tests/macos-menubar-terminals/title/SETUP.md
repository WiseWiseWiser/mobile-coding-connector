# Scenario

**Feature**: terminal session submenu title strings (name/id base + optional exited suffix)

```
# base = non-empty trimmed name, else id
name + id + status -> FormatTerminalTitle -> display title
# status exited (case-insensitive, trim) -> base + " [EXITED]"
# status running / empty / unknown -> base only
```

## Preconditions

`Op=title` dispatches to `menubar.FormatTerminalTitle(name, id, status)`.

## Steps

1. Leaf supplies `Name`, `SessionID`, and `Status`.

## Context

REQUIREMENT: base title is name if set else id; exited sessions append exact
` [EXITED]`. Cleared is not covered (server removes from list).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "title"
	return nil
}
```
