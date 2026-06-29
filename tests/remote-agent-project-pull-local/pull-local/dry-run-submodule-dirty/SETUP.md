# Scenario

**Feature**: dry-run still runs submodule guard and fails before plan

```
# dirty submodule + --dry-run
exit 1 without dry-run plan / worktree
```

## Preconditions

Dirty submodule; binding present.

## Steps

1. Submodule repo with dirty `submod/`.
2. Args: `project pull-local <id> --dry-run`.

## Context

REQUIREMENT leaf `pull-local/dry-run-submodule-dirty`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	pair := pairSameOriginWithSubmodule(t)
	remoteDir, localDir := pair.RemoteDir, pair.LocalDir
	dirtySubmoduleFile(t, remoteDir)
	registerPullProject(t, req, "pull-drysub-001", "pull-drysub", remoteDir)
	seedBindingForServer(t, req, remoteDir, localDir)
	req.Args = []string{"project", "pull-local", "pull-drysub-001", "--dry-run"}
	return nil
}
```