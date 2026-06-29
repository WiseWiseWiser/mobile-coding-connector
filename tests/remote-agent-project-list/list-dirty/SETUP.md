# Scenario

**Feature**: `remote-agent project list --dirty` filters to dirty worktrees only

```
# register clean + dirty projects -> project list --dirty -> only dirty printed
```

## Preconditions

- Harness supports multiple projects in `projects.json`.

## Steps

1. Descendant leaves create one or more project directories and set `req.Projects`.
2. Set `req.Args` to `[]string{"project", "list", "--dirty"}`.

## Context

- Non-git projects are never listed with `--dirty`.
- Clean git repos are omitted.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Args = []string{"project", "list", "--dirty"}
	return nil
}
```