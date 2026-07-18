# Scenario

**Feature**: CLI exclude adds a custom path

```
MergeExclusions(nil, [".docker"], nil) -> IsExcluded(".docker") == true
  reason user excluded
```

## Preconditions

- `.docker` is not a builtin catalog path.

## Steps

1. Exclude `.docker`.
2. Expect excluded with reason.

## Context

- Custom excludes remain after pathflag SSoT for builtin catalog.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Exclude = []string{".docker"}
	req.Include = nil
	req.RelPath = ".docker"
	req.WantExcluded = true
	req.WantExcludedSet = true
	return nil
}
```
