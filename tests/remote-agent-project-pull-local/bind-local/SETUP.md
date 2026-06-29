# Scenario

**Feature**: `remote-agent project bind-local` validates origins and persists bindings

```
# resolve project, compare local vs remote origin via exec, upsert project_bindings
remote-agent project bind-local <remote-dir> <local-path> -> remote-agent-config.json
```

## Preconditions

Remote project is registered in `projects.json`. Local path exists.

## Steps

1. Leaf creates remote + local git layout (shared or divergent bare origins).
2. `Run` invokes `project bind-local` with resolved paths.
3. `Assert` checks exit code, stderr on failure, and `project_bindings` on success.

## Context

Grouping node for bind-local MECE branches: same origin, origin mismatch, not a git repo.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if len(req.Args) >= 2 && req.Args[1] != "bind-local" {
		t.Fatalf("bind-local group: unexpected subcommand argv %v", req.Args)
	}
	return nil
}
```