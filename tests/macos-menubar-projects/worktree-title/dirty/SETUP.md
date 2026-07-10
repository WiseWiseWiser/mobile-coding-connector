# Scenario

**Feature**: dirty worktree title

```
FormatWorktreeTitle("feat-login", false) -> "feat-login ○ Dirty"
```

## Preconditions

Linked worktree basename `feat-login` is dirty.

## Steps

1. Set name `feat-login`, clean `false`.

## Context

REQUIREMENT leaf: worktree title dirty.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Name = "feat-login"
	req.Clean = false
	return nil
}
```
