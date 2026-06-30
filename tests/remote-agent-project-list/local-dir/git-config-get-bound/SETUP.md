# Scenario

**Feature**: git-config get prints Local Dir for bound project

```
# binding + project git-config get <id> -> same Local Dir line as list
```

## Preconditions

- Binding seeded for registered project.

## Steps

1. Set `req.Args` to `project git-config get <id>`.

## Context

REQUIREMENT scope: `printProjectGitConfig` via git-config get.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	dir := mkProjectDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")
	localDir := mkLocalBindingDir(t)
	req.Project = ProjectEntry{ID: "local-gcfg-001", Name: "local-git-config", Dir: dir}
	seedListBinding(t, req, dir, localDir)
	req.Args = []string{"project", "git-config", "get", "local-gcfg-001"}
	return nil
}
```