# Scenario

**Feature**: terminal session submenu title strings

```
name + id -> FormatTerminalTitle -> display title (name if set, else id)
```

## Preconditions

`Op=title` dispatches to `menubar.FormatTerminalTitle`.

## Steps

1. Leaf supplies `Name` and `SessionID`.

## Context

REQUIREMENT: session title is name-only; empty/whitespace name falls back to id.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "title"
	return nil
}
```
