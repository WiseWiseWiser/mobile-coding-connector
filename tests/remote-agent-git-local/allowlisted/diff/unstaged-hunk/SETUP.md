# Scenario

**Feature**: unstaged diff shows hunk

```
modify tracked.txt -> diff -> @@ hunk
```

## Preconditions

Committed `tracked.txt`.

## Steps

1. Overwrite `tracked.txt` after commit.
2. Run `diff` (default: unstaged).

## Context

REQUIREMENT leaf #3.

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
	if err := os.WriteFile(path, []byte("before\n"), 0644); err != nil {
		return err
	}
	gitRun(t, dir, "add", "tracked.txt")
	gitRun(t, dir, "commit", "-m", "Initial commit")
	if err := os.WriteFile(path, []byte("after\n"), 0644); err != nil {
		return err
	}
	setGitLocalArgs(t, req, dir, "diff", "tracked.txt")
	return nil
}
```