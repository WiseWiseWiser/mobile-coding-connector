# Scenario

**Feature**: --dirty list still shows Local Dir for bound dirty project

```
# binding + dirty repo -> project list --dirty -> Local Dir on dirty project only
```

## Preconditions

- Binding matches dirty project's server `Dir`.

## Steps

1. Dirty git repo with binding.
2. `req.Args` = `project list --dirty`.

## Context

REQUIREMENT leaf `local-dir/bound-dirty-filter`.

```go
import (
	"os"
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir := mkProjectDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")
	if err := os.WriteFile(filepath.Join(dir, "dirty-only.txt"), []byte("x\n"), 0644); err != nil {
		return err
	}
	localDir := mkLocalBindingDir(t)
	req.Project = ProjectEntry{ID: "local-dirty-001", Name: "local-dirty-bound", Dir: dir}
	seedListBinding(t, req, dir, localDir)
	req.Args = []string{"project", "list", "--dirty"}
	return nil
}
```