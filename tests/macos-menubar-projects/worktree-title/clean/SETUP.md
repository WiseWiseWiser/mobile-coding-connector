# Scenario

**Feature**: clean worktree title parts

```
FormatWorktreeTitleParts("feat-login", true) -> Leading="feat-login", Trailing="● Clean"
FormatWorktreeTitle(...) -> "feat-login  ● Clean"
```

## Preconditions

Linked worktree basename `feat-login` is clean.

## Steps

1. Set name `feat-login`, clean `true`.

## Context

REQUIREMENT: worktree clean → `● Clean`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Name = "feat-login"
	req.Clean = true
	return nil
}
```
