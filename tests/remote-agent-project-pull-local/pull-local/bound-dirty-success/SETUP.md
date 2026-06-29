# Scenario

**Feature**: pull-local with binding transfers dirty state and cleans remote

```
# seeded project_bindings, modified README + untracked file on remote
remote-agent project pull-local <id> -> worktree with changes, remote porcelain empty
```

## Preconditions

Binding for `(server, remote_dir)`; remote dirty; local clone same origin.

## Steps

1. `pairSameOriginRepos`; dirty top-level.
2. Seed binding; register `pull-bound-001`.
3. Args: `project pull-local pull-bound-001`.

## Context

REQUIREMENT leaf `pull-local/bound-dirty-success`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	pair := pairSameOriginRepos(t)
	remoteDir, localDir := pair.RemoteDir, pair.LocalDir
	dirtyTopLevelModifiedAndUntracked(t, remoteDir)
	registerPullProject(t, req, "pull-bound-001", "pull-bound", remoteDir)
	seedBindingForServer(t, req, remoteDir, localDir)
	req.Args = []string{"project", "pull-local", "pull-bound-001"}
	return nil
}
```