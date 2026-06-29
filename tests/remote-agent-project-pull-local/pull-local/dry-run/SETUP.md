# Scenario

**Feature**: --dry-run prints plan without mutations

```
# dirty remote, binding, --dry-run
stdout plan; no worktree; remote still dirty
```

## Preconditions

Dirty remote and binding.

## Steps

1. Dirty remote; seed binding.
2. Args: `project pull-local <id> --dry-run`.

## Context

REQUIREMENT leaf `pull-local/dry-run`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	pair := pairSameOriginRepos(t)
	remoteDir, localDir := pair.RemoteDir, pair.LocalDir
	dirtyTopLevelModifiedAndUntracked(t, remoteDir)
	registerPullProject(t, req, "pull-dry-001", "pull-dry", remoteDir)
	seedBindingForServer(t, req, remoteDir, localDir)
	req.Args = []string{"project", "pull-local", "pull-dry-001", "--dry-run"}
	return nil
}
```