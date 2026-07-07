# Scenario

**Feature**: restore --show-meta prints git-repo-worktrees.json section

```
# prereq backup with git fixtures -> restore --show-meta -> git JSON section
```

## Preconditions

`git` on PATH; prereq backup from `SeedGitRepos` server home (custom archive).

## Steps

1. `SeedGitRepos=true`, `ShowMeta=true`.
2. Args: `machine restore` (archive injected by Run).

## Context

REQUIREMENT leaf `restore/show-meta-git-repos`. Complements `restore/show-meta`
(installed.json + ENV only).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	requireGit(t)
	req.SeedGitRepos = true
	req.ShowMeta = true
	req.Args = []string{"machine", "restore"}
	return nil
}
```