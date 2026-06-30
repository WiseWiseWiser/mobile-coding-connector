# Scenario

**Feature**: bound project shows absolute Local Dir on list

```
# project_bindings (server, remote_dir) -> list -> Local Dir: <local_path>
```

## Preconditions

- One clean git project registered on the server.

## Steps

1. Create remote project repo with initial commit.
2. Seed binding to a separate temp local path.
3. Run `project list`.

## Context

REQUIREMENT leaf `local-dir/bound`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	dir := mkProjectDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")
	localDir := mkLocalBindingDir(t)
	req.Project = ProjectEntry{ID: "local-bound-001", Name: "local-bound", Dir: dir}
	seedListBinding(t, req, dir, localDir)
	return nil
}
```