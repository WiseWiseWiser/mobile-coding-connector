# Scenario

**Feature**: staged diff via `diff --cached`

```
stage change -> diff --cached -> staged hunk only
```

## Preconditions

One commit; staged modification.

## Steps

1. Commit `file.txt`.
2. Modify and `git add` file.
3. Run `diff --cached`.

## Context

Requirement notes `diff --cached`.

```go
import (
	"os"
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir := mkWorkDir(t)
	gitInitWithMain(t, dir)
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("v1\n"), 0644); err != nil {
		return err
	}
	gitRun(t, dir, "add", "file.txt")
	gitRun(t, dir, "commit", "-m", "Initial commit")
	if err := os.WriteFile(path, []byte("v2\n"), 0644); err != nil {
		return err
	}
	gitRun(t, dir, "add", "file.txt")
	setGitLocalArgs(t, req, dir, "diff", "--cached")
	return nil
}
```