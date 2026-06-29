# Scenario

**Feature**: --dirty with only clean projects prints no-dirty message

```
# one clean repo -> project list --dirty -> "No dirty projects found."
```

## Preconditions

- Single clean git project registered.

## Steps

1. Create clean repo with initial commit.
2. Register project and run with `--dirty`.

## Context

- Distinct from the global empty `No projects found.` case.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	dir := mkProjectDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")

	req.Projects = []ProjectEntry{
		{ID: "clean-only-001", Name: "clean-only", Dir: dir},
	}
	return nil
}
```