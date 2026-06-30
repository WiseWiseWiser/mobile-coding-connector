# Scenario

**Feature**: binding for a different server does not match list --server

```
# binding server http://other:1 -> list against test server -> Local Dir: -
```

## Preconditions

- Registered project on live test server.

## Steps

1. Seed binding with `Server: http://127.0.0.1:1` (not the CLI `--server`).

## Context

REQUIREMENT leaf `local-dir/wrong-server`.

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
	absRemote, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	absLocal, err := filepath.Abs(localDir)
	if err != nil {
		return err
	}
	req.Project = ProjectEntry{ID: "local-wsrv-001", Name: "local-wrong-server", Dir: dir}
	req.SeedBindings = []ProjectBinding{{
		Server:    "http://127.0.0.1:1",
		RemoteDir: absRemote,
		LocalPath: absLocal,
	}}
	return nil
}
```