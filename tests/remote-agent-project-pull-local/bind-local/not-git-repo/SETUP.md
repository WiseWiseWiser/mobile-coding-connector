# Scenario

**Feature**: bind-local requires a local git repository

```
# local path is a plain directory without .git
remote-agent project bind-local -> not a git repository error
```

## Preconditions

Remote project is a valid git repo; local path has no `.git`.

## Steps

1. Same-origin remote clone; local dir is empty mkdir without `git init`.
2. Register project and run bind-local.

## Context

REQUIREMENT leaf `bind-local/not-git-repo`.

```go
import (
	"os"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	remoteDir := pairSameOriginRepos(t).RemoteDir
	localPlain := mkLocalDir(t)
	if err := os.WriteFile(localPlain+"/note.txt", []byte("not git\n"), 0644); err != nil {
		return err
	}
	registerPullProject(t, req, "bind-notgit-001", "bind-notgit", remoteDir)
	req.LocalPath = localPlain
	req.Args = []string{"project", "bind-local", remoteDir, localPlain}
	return nil
}
```