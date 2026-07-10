# Scenario

**Feature**: linked worktree row title parts (Leading left / Trailing right)

```
# basename + clean -> FormatWorktreeTitleParts -> {Leading, Trailing}
name, clean -> FormatWorktreeTitleParts -> Leading, Trailing
legacy FormatWorktreeTitle -> Leading + "  " + Trailing
```

## Preconditions

`Op=worktree_title` dispatches to `menubar.FormatWorktreeTitleParts` and legacy
`FormatWorktreeTitle`.

## Steps

1. Leaf supplies worktree `Name` (basename) and `Clean`.

## Context

REQUIREMENT scenario 4 (worktree clean/dirty).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "worktree_title"
	return nil
}
```
