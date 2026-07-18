# Scenario

**Feature**: machinebackup exclusions integrate pathflag catalog as SSoT

```
# module dependency
go.mod -> require/replace github.com/xhd2015/bak-files

# package wiring
server/machinebackup -> import pathflag -> Classify / DefaultSkipMask

# policy merge (public API)
MergeExclusions(user, exclude, include) -> IsExcluded / ReasonFor
BuiltinExclusionConfig() -> exclude_paths aligned with pathflag catalog + **(binary)
```

## Preconditions

- Module root: `DOCTEST_ROOT/../..` (ai-critic snapshot with `go.mod`).
- Production package: `github.com/xhd2015/ai-critic/server/machinebackup`.
- Catalog library (post-refactor): `github.com/xhd2015/bak-files/pathflag`.
- Classic TDD: go.mod does not yet require bak-files; machinebackup does not
  yet import pathflag — module/import leaves must RED until implementer lands.
- Pure exclusion leaves call only public APIs (`MergeExclusions`, `IsExcluded`,
  `ReasonFor`, `BuiltinExclusionConfig`). No production edits from this tree.
- Default `Request.Op` is `is_excluded` when unset.

## Steps

1. Root Setup defaults `Op` to `is_excluded` when empty.
2. Grouping / leaf Setup set Op, RelPath, Include/Exclude, and expectation flags.
3. Root `Run` dispatches to machinebackup or go.mod / import inspection.
4. Leaf Assert checks Response fields against post-refactor contract.

## Context

- Skip model after refactor: pathflag flags ∩ DefaultSkipMask, plus user/CLI
  excludes, minus includes; binary content detect remains filesystem-based.
- Log suffix (`**/*.log`) is pathflag Classify behavior; public `IsExcluded`
  must reflect it after wiring (today only walk-time `shouldSkipPath` applies logs).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.Op == "" {
		req.Op = OpIsExcluded
	}
	return nil
}
```
