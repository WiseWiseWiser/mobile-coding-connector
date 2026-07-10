# Scenario

**Feature**: clicking a worktree row opens worktree.path with reuse

```
worktree Button -> openITerm2(dir: wt.path, mode: reuse)
```

## Preconditions

Projects submenu lists worktrees.

## Steps

1. Set `ClientLeaf=click-worktree-opens`.

## Context

REQUIREMENT: click worktree → open worktree.path reuse.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "click-worktree-opens"
	return nil
}
```
