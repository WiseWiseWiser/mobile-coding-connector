# Scenario

**Feature**: binding remote_dir mismatch yields Local Dir dash

```
# binding points at other path -> list project -> Local Dir: -
```

## Preconditions

- Project `Dir` differs from binding `remote_dir`.

## Steps

1. Seed binding with a different `RemoteDir` than registered project.

## Context

REQUIREMENT leaf `local-dir/wrong-remote-dir`.

```go
import (
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir := mkProjectDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")
	localDir := mkLocalBindingDir(t)
	otherRemote := mkProjectDir(t)
	absLocal, err := filepath.Abs(localDir)
	if err != nil {
		return err
	}
	absOther, err := filepath.Abs(otherRemote)
	if err != nil {
		return err
	}
	req.Project = ProjectEntry{ID: "local-wdir-001", Name: "local-wrong-dir", Dir: dir}
	req.SeedBindings = []ProjectBinding{{
		RemoteDir: absOther,
		LocalPath: absLocal,
	}}
	return nil
}
```