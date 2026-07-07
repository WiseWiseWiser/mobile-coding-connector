# Scenario

**Feature**: dry-run GIT REPOS nests linked worktree with dirty status

```
# main repo + linked worktree with dirty file -> backup --dry-run -> nested worktree block
worktree .wrk-test/feature-wt with dirty (N modified) count
```

## Preconditions

`git` on PATH; `SeedGitReposWorktree` seeds main repo and dirty worktree checkout.

## Steps

1. `SeedGitReposWorktree=true`.
2. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/git-repos-worktree`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	requireGit(t)
	req.SeedGitReposWorktree = true
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```