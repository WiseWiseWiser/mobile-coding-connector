# Scenario

**Feature**: --dirty lists only the dirty project when a clean project is also registered

```
# clean repo + dirty repo registered -> --dirty -> stdout has dirty only
```

## Preconditions

- Two projects in `projects.json`.

## Steps

1. Create clean repo with initial commit.
2. Create dirty repo with one untracked file after commit.
3. Register both projects.

## Context

- Clean project must not appear in stdout.

```go
import (
	"os"
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	cleanDir := mkProjectDir(t)
	gitInitWithMain(t, cleanDir)
	gitInitialCommit(t, cleanDir, "Initial commit")

	dirtyDir := mkProjectDir(t)
	gitInitWithMain(t, dirtyDir)
	gitInitialCommit(t, dirtyDir, "Initial commit")
	if err := os.WriteFile(filepath.Join(dirtyDir, "dirty.txt"), []byte("dirty\n"), 0644); err != nil {
		return err
	}

	req.Projects = []ProjectEntry{
		{ID: "clean-001", Name: "clean-project", Dir: cleanDir},
		{ID: "dirty-001", Name: "dirty-project", Dir: dirtyDir},
	}
	return nil
}
```