# Scenario

**Feature**: list shows Local Dir dash when no binding exists

```
# empty project_bindings -> list -> Local Dir: -
```

## Preconditions

- Project registered; no `SeedBindings` on `Request`.

## Steps

1. Clean git repo; do not seed bindings.

## Context

REQUIREMENT leaf `local-dir/unbound`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	dir := mkProjectDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")
	req.Project = ProjectEntry{ID: "local-unbound-001", Name: "local-unbound", Dir: dir}
	req.SeedBindings = nil
	return nil
}
```