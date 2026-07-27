# Scenario

**Feature**: bind-local saves binding when origins match

```
# same file:// bare origin on remote project dir and local clone
remote-agent project bind-local <remote-dir> <local-path> -> project_bindings upsert
```

## Preconditions

Remote and local repos cloned from the same bare origin.

## Steps

1. `pairSameOriginRepos` creates remote project dir and local clone.
2. Register project `bind-same` / `bind-same-001`.
3. Args: `project bind-local <remoteDir> <localDir>`.

## Context

REQUIREMENT leaf `bind-local/same-origin`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	pair := pairSameOriginRepos(t)
	remoteDir, localDir := pair.RemoteDir, pair.LocalDir
	registerPullProject(t, req, "bind-same-001", "bind-same", remoteDir)
	req.LocalPath = localDir
	req.Args = []string{"project", "bind-local", remoteDir, localDir}
	return nil
}
```