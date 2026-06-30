# Scenario

**Feature**: git-config get shows Local Dir dash without binding

```
# no project_bindings -> project git-config get -> Local Dir: -
```

## Preconditions

- Registered project; empty bindings in agent config.

## Steps

1. Args: `project git-config get <id>`.

## Context

REQUIREMENT: git-config get uses `printProjectGitConfig`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	dir := mkProjectDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")
	req.Project = ProjectEntry{ID: "local-gcfg-u-001", Name: "local-git-config-unbound", Dir: dir}
	req.Args = []string{"project", "git-config", "get", "local-gcfg-u-001"}
	return nil
}
```