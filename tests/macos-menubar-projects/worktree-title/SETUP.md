# Scenario

**Feature**: linked worktree row title strings

```
basename + clean -> FormatWorktreeTitle -> row title
```

## Preconditions

`Op=worktree_title` dispatches to `menubar.FormatWorktreeTitle`.

## Steps

1. Leaf supplies worktree `Name` (basename) and `Clean`.

## Context

REQUIREMENT scenario 16.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "worktree_title"
	return nil
}
```
