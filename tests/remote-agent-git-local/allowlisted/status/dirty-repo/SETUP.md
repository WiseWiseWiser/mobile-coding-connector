# Scenario

**Feature**: dirty repo status lists modified and untracked files

```
modify tracked + add untracked -> status mentions both paths
```

## Preconditions

One commit baseline.

## Steps

1. Commit `tracked.txt`.
2. Modify `tracked.txt`; create `untracked.txt`.

## Context

REQUIREMENT leaf #2.

```go
import (
	"os"
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir := mkWorkDir(t)
	gitInitWithMain(t, dir)
	path := filepath.Join(dir, "tracked.txt")
	if err := os.WriteFile(path, []byte("v1\n"), 0644); err != nil {
		return err
	}
	gitRun(t, dir, "add", "tracked.txt")
	gitRun(t, dir, "commit", "-m", "Initial commit")
	if err := os.WriteFile(path, []byte("v2\n"), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("new\n"), 0644); err != nil {
		return err
	}
	setGitLocalArgs(t, req, dir, "status")
	return nil
}
```