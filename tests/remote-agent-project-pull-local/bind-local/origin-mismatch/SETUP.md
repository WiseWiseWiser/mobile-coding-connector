# Scenario

**Feature**: bind-local rejects different git origins

```
# remote and local clones from different bare repos
remote-agent project bind-local -> origin mismatch error
```

## Preconditions

Two independent bare origins.

## Steps

1. `pairMismatchedOriginRepos`.
2. Register `bind-mismatch-001` / `bind-mismatch`.
3. Run bind-local with both paths.

## Context

REQUIREMENT leaf `bind-local/origin-mismatch`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	pair := pairMismatchedOriginRepos(t)
	remoteDir, localDir := pair.RemoteDir, pair.LocalDir
	registerPullProject(t, req, "bind-mismatch-001", "bind-mismatch", remoteDir)
	req.LocalPath = localDir
	req.Args = []string{"project", "bind-local", remoteDir, localDir}
	return nil
}
```