# Scenario

**Feature**: nested `upload-chunks` segment is excluded

```
MergeExclusions(nil,nil,nil) -> IsExcluded("a/upload-chunks/1") == true
  rule **/upload-chunks
```

## Preconditions

- Segment rule for incomplete upload temp state.

## Steps

1. RelPath with `upload-chunks` component.
2. Expect excluded.

## Context

- pathflag FlagTmp under DefaultSkipMask.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.RelPath = "a/upload-chunks/1"
	req.WantExcluded = true
	req.WantExcludedSet = true
	return nil
}
```
