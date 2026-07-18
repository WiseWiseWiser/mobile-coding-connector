# Scenario

**Feature**: nested `node_modules` segment is excluded

```
MergeExclusions(nil,nil,nil) -> IsExcluded("foo/node_modules/x") == true
  rule **/node_modules
```

## Preconditions

- Segment rule when no longer path-prefix catalog match.

## Steps

1. RelPath with `node_modules` component.
2. Expect excluded.

## Context

- pathflag segment Vendor → DefaultSkipMask.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.RelPath = "foo/node_modules/x"
	req.WantExcluded = true
	req.WantExcludedSet = true
	return nil
}
```
