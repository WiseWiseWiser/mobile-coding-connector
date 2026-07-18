# Scenario

**Feature**: CLI include/exclude policy overrides on top of catalog

```
# include removes builtin exclude
MergeExclusions(nil, nil, include) -> IsExcluded(.cache/...) == false

# exclude adds custom path
MergeExclusions(nil, exclude, nil) -> IsExcluded(.docker) == true

# include specific log keeps it; other logs still skip
MergeExclusions(nil, nil, [keep.log]) -> keep not excluded; service.log excluded
```

## Preconditions

- Effective rule: (defaults − include) ∪ exclude; CLI exclude wins over include.

## Steps

1. Group uses OpIncludeOverride (same Run path as exclusion with slices).
2. Leaves set Include/Exclude and paths.

## Context

- Regression coverage may be GREEN pre-refactor when only path tables apply;
  log interaction stays RED until IsExcluded honors pathflag log suffix.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpIncludeOverride
	return nil
}
```
