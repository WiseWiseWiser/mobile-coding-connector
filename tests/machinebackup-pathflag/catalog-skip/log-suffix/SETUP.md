# Scenario

**Feature**: basename `*.log` is catalog-excluded via public IsExcluded

```
MergeExclusions(nil,nil,nil) -> IsExcluded(".ai-critic/service.log") == true
  rule **/*.log (pathflag FlagLogs / DefaultSkipMask)
```

## Preconditions

- Post-refactor: Classify log suffix drives IsExcluded without needing file mode.
- Pre-refactor: only walk-time shouldSkipPath applies `**/*.log` — public
  IsExcluded does not — this leaf is intentionally RED today.

## Steps

1. RelPath `.ai-critic/service.log`.
2. Expect excluded with non-empty reason.

## Context

- Forces wiring of pathflag log suffix into ExclusionRules public API.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.RelPath = ".ai-critic/service.log"
	req.WantExcluded = true
	req.WantExcludedSet = true
	return nil
}
```
