# Scenario

**Feature**: second pull allocates main-2 when main-1 exists

```
# pull-local twice with re-dirty between runs (WorktreeCollision in Run)
second worktree path ends with main-2
```

## Preconditions

Binding; first pull succeeds and truncates remote; remote re-dirtied; second pull succeeds.

## Steps

1. Dirty remote + binding.
2. `WorktreeCollision` true → Run executes two identical pull-local argv.

## Context

REQUIREMENT leaf `pull-local/worktree-collision`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	pair := pairSameOriginRepos(t)
	remoteDir, localDir := pair.RemoteDir, pair.LocalDir
	dirtyTopLevelModifiedAndUntracked(t, remoteDir)
	registerPullProject(t, req, "pull-collision-001", "pull-collision", remoteDir)
	seedBindingForServer(t, req, remoteDir, localDir)
	req.WorktreeCollision = true
	req.Args = []string{"project", "pull-local", "pull-collision-001"}
	return nil
}
```