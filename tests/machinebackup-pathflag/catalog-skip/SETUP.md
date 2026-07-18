# Scenario

**Feature**: catalog skip aligned with pathflag DefaultSkipMask paths

```
MergeExclusions(nil, nil, nil) -> IsExcluded(rel) / ReasonFor(rel)
# skip when pathflag would set Flags & DefaultSkipMask != 0
# include ordinary configs (.bashrc, .codex/config.toml)
```

## Preconditions

- No user config; no CLI exclude/include.
- Public API only — harness does not call unexported shouldSkipPath.

## Steps

1. Group forces empty Exclude/Include.
2. Leaves set RelPath and expected excluded outcome.

## Context

- After refactor, log suffix is path-classifiable; IsExcluded must honor it.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpIsExcluded
	req.Exclude = nil
	req.Include = nil
	return nil
}
```
