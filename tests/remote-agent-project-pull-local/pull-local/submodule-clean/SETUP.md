# Scenario

**Feature**: pull-local succeeds when submodule is clean but top-level is dirty

```
# .gitmodules present, submodule clean, parent dirty
remote-agent project pull-local -> exit 0
```

## Preconditions

Initialized submodule; dirty only at repository root.

## Steps

1. `pairSameOriginWithSubmodule`.
2. Dirty top-level README; binding seeded.

## Context

REQUIREMENT leaf `pull-local/submodule-clean`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	pair := pairSameOriginWithSubmodule(t)
	remoteDir, localDir := pair.RemoteDir, pair.LocalDir
	dirtyTopLevelModifiedAndUntracked(t, remoteDir)
	registerPullProject(t, req, "pull-subclean-001", "pull-subclean", remoteDir)
	seedBindingForServer(t, req, remoteDir, localDir)
	req.Args = []string{"project", "pull-local", "pull-subclean-001"}
	return nil
}
```