# Scenario

**Feature**: successful pull leaves worktree on a named branch (not detached)

```
# pull-local -> git worktree add -b <suffix> -> symbolic-ref HEAD + branch --show-current
```

## Preconditions

- Same-origin binding and dirty remote (same as bound-dirty-success).

## Steps

1. Dirty remote + binding; `project pull-local` by id.

## Context

REQUIREMENT leaf `pull-local/worktree-on-branch`. Branch name must equal worktree
directory basename (`main-1`).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	pair := pairSameOriginRepos(t)
	remoteDir, localDir := pair.RemoteDir, pair.LocalDir
	dirtyTopLevelModifiedAndUntracked(t, remoteDir)
	registerPullProject(t, req, "pull-branch-001", "pull-branch", remoteDir)
	seedBindingForServer(t, req, remoteDir, localDir)
	req.Args = []string{"project", "pull-local", "pull-branch-001"}
	return nil
}
```