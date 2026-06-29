# Scenario

**Feature**: pull-local refuses a clean remote worktree

```
# remote repo clean, binding present
remote-agent project pull-local -> nothing to pull
```

## Preconditions

Clean remote; binding seeded.

## Steps

1. Same-origin pair without dirty changes.
2. Seed binding; run pull-local by project id.

## Context

REQUIREMENT leaf `pull-local/clean-remote`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	pair := pairSameOriginRepos(t)
	remoteDir, localDir := pair.RemoteDir, pair.LocalDir
	registerPullProject(t, req, "pull-clean-001", "pull-clean", remoteDir)
	seedBindingForServer(t, req, remoteDir, localDir)
	req.Args = []string{"project", "pull-local", "pull-clean-001"}
	return nil
}
```