# Scenario

**Feature**: `.cache` tree is catalog-excluded

```
MergeExclusions(nil,nil,nil) -> IsExcluded(".cache/x") == true
  ReasonFor -> temporary application cache (pathflag .cache)
```

## Preconditions

- Path catalog rule `.cache` → FlagCache under DefaultSkipMask.

## Steps

1. RelPath `.cache/x`.
2. Expect excluded with non-empty reason.

## Context

- Representative pathflag cache rule under home.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.RelPath = ".cache/x"
	req.WantExcluded = true
	req.WantExcludedSet = true
	return nil
}
```
