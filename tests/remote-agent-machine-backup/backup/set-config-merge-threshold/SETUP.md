# Scenario

**Feature**: exclude-only --set-config preserves persisted large_dir_threshold

```
# prereq set-config .knowledge-hub + 50MB threshold, then set-config --exclude .docker only
persisted file has both excludes and large_dir_threshold 50MB
```

## Preconditions

`serverHome` includes `.knowledge-hub` fixtures (`SeedKnowledgeHub`).

## Steps

1. `SeedKnowledgeHub=true`, `PrereqSetConfig=true`, `SetConfigExcludePaths=[".knowledge-hub"]`, `PrereqSetConfigLargeDirThreshold="50MB"`.
2. Main invocation: `SetConfig=true`, `ExcludePaths=[".docker"]` (no threshold flag).
3. Args: `machine backup`.

## Context

Set-config merge leaf: exclude-only invocation must not wipe prior threshold.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedKnowledgeHub = true
	req.PrereqSetConfig = true
	req.SetConfigExcludePaths = []string{".knowledge-hub"}
	req.PrereqSetConfigLargeDirThreshold = "50MB"
	req.SetConfig = true
	req.ExcludePaths = []string{".docker"}
	req.Args = []string{"machine", "backup"}
	return nil
}
```