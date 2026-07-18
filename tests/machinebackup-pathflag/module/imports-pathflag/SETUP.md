# Scenario

**Feature**: machinebackup package imports bak-files/pathflag

```
go list github.com/xhd2015/ai-critic/server/machinebackup
  -> Imports includes github.com/xhd2015/bak-files/pathflag
```

## Preconditions

- Prefer `go list -f '{{join .Imports "\n"}}'`; source parse is fallback.

## Steps

1. Set Op to package_imports.
2. Expect ImportsPathflag true.

## Context

- RED until implementer adds the import and uses Classify for catalog skip.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpPackageImports
	return nil
}
```
