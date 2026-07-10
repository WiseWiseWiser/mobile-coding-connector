# Scenario

**Feature**: clean worktree title

```
FormatWorktreeTitle("feat-login", true) -> "feat-login ● Clean"
```

## Preconditions

Linked worktree basename `feat-login` is clean.

## Steps

1. Set name `feat-login`, clean `true`.

## Context

REQUIREMENT leaf: worktree title clean.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Name = "feat-login"
	req.Clean = true
	return nil
}
```
