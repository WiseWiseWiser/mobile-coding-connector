# Scenario

**Feature**: dirty worktree title parts

```
FormatWorktreeTitleParts("feat-login", false) -> Leading="feat-login", Trailing="○ Dirty"
FormatWorktreeTitle(...) -> "feat-login  ○ Dirty"
```

## Preconditions

Linked worktree basename `feat-login` is dirty.

## Steps

1. Set name `feat-login`, clean `false`.

## Context

REQUIREMENT: worktree dirty → `○ Dirty`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Name = "feat-login"
	req.Clean = false
	return nil
}
```
