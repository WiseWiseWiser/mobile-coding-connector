# Scenario

**Feature**: pull-local fails when a submodule worktree is dirty

```
# dirty file inside submod/
remote-agent project pull-local -> error names submodule path
```

## Preconditions

Submodule initialized; dirty inside `submod/`.

## Steps

1. `pairSameOriginWithSubmodule`.
2. `dirtySubmoduleFile` on remote.
3. Binding seeded.

## Context

REQUIREMENT leaf `pull-local/submodule-dirty`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	pair := pairSameOriginWithSubmodule(t)
	remoteDir, localDir := pair.RemoteDir, pair.LocalDir
	dirtySubmoduleFile(t, remoteDir)
	registerPullProject(t, req, "pull-subdirty-001", "pull-subdirty", remoteDir)
	seedBindingForServer(t, req, remoteDir, localDir)
	req.Args = []string{"project", "pull-local", "pull-subdirty-001"}
	return nil
}
```