# Scenario

**Feature**: include a specific log file; other logs still skip

```
MergeExclusions(nil, nil, [".ai-critic/keep.log"])
  -> IsExcluded(".ai-critic/keep.log") == false
  -> IsExcluded(".ai-critic/service.log") == true  # **/*.log still applies
```

## Preconditions

- Exact-path include override for keep.log.
- service.log remains under log suffix catalog skip (public API post-refactor).

## Steps

1. Include only `.ai-critic/keep.log`.
2. Primary RelPath = keep.log (not excluded).
3. SecondaryRelPath = service.log (excluded).

## Context

- Secondary assertion is RED until IsExcluded honors pathflag `**/*.log`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Include = []string{".ai-critic/keep.log"}
	req.Exclude = nil
	req.RelPath = ".ai-critic/keep.log"
	req.WantExcluded = false
	req.WantExcludedSet = true
	req.SecondaryRelPath = ".ai-critic/service.log"
	req.WantSecondaryExcluded = true
	req.WantSecondaryExcludedSet = true
	return nil
}
```
