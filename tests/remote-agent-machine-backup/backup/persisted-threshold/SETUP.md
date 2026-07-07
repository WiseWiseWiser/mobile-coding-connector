# Scenario

**Feature**: persisted large_dir_threshold applies when CLI omits threshold

```
# prereq set-config 100MB, dry-run without CLI threshold -> 50MB dir lacks LARGE SIZE
persisted threshold overrides 40MB default
```

## Preconditions

`SeedLargeDir` writes `.big-test/` totaling ~50 MB.

## Steps

1. `SeedLargeDir=true`, `PrereqSetConfig=true`, `PrereqSetConfigLargeDirThreshold="100MB"`.
2. Args: `machine backup --dry-run` (no `--large-dir-threshold`).

## Context

Runtime threshold resolution from persisted user config.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedLargeDir = true
	req.PrereqSetConfig = true
	req.PrereqSetConfigLargeDirThreshold = "100MB"
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```