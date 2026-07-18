# Scenario

**Feature**: module and package dependency contract on bak-files/pathflag

```
# go.mod must require and replace bak-files
go.mod -> require github.com/xhd2015/bak-files
go.mod -> replace github.com/xhd2015/bak-files => ../..

# machinebackup must import pathflag
server/machinebackup -> import pathflag
```

## Preconditions

- Snapshot lives at `external/ai-critic-master-…` so replace path is `../..`.
- Inspection uses file/`go list` only — does not import pathflag into the test binary
  (so catalog leaves can compile before the dependency lands).

## Steps

1. Group marks module-contract scope (leaves set concrete Op).
2. Leaves assert require/replace or Imports.

## Context

- These leaves are the primary RED-forcing contract for classic TDD.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// Clear path fields so module leaves never accidentally hit exclusion ops.
	req.RelPath = ""
	req.RulePath = ""
	return nil
}
```
