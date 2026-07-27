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
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Exclude = []string{".docker"}
	req.Include = nil
	req.RelPath = ".docker"
	req.WantExcluded = true
	req.WantExcludedSet = true
	return nil
}
```
