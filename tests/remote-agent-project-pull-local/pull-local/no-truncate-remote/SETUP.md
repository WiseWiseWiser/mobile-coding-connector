# Scenario

**Feature**: --no-truncate-remote keeps remote dirty after pull

```
# bound dirty remote, pull-local --no-truncate-remote
local worktree created; remote porcelain unchanged
```

## Preconditions

Binding and dirty remote.

## Steps

1. Same-origin dirty remote.
2. Args include `--no-truncate-remote`.

## Context

REQUIREMENT leaf `pull-local/no-truncate-remote`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	pair := pairSameOriginRepos(t)
	remoteDir, localDir := pair.RemoteDir, pair.LocalDir
	dirtyTopLevelModifiedAndUntracked(t, remoteDir)
	registerPullProject(t, req, "pull-notrunc-001", "pull-notrunc", remoteDir)
	seedBindingForServer(t, req, remoteDir, localDir)
	req.Args = []string{"project", "pull-local", "pull-notrunc-001", "--no-truncate-remote"}
	return nil
}
```