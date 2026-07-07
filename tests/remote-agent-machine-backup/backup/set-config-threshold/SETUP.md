# Scenario

**Feature**: threshold-only --set-config persists large_dir_threshold without wiping prior excludes

```
# prereq set-config .knowledge-hub, then set-config --large-dir-threshold 100MB only
persisted file keeps exclude_paths and updates threshold
```

## Preconditions

`serverHome` includes `.knowledge-hub` fixtures (`SeedKnowledgeHub`).

## Steps

1. `SeedKnowledgeHub=true`, `PrereqSetConfig=true`, `SetConfigExcludePaths=[".knowledge-hub"]`.
2. Main invocation: `SetConfig=true`, `SetConfigLargeDirThreshold="100MB"` (no `--exclude`).
3. Args: `machine backup`.

## Context

Threshold-only set-config must merge threshold into existing persisted config.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedKnowledgeHub = true
	req.PrereqSetConfig = true
	req.SetConfigExcludePaths = []string{".knowledge-hub"}
	req.SetConfig = true
	req.SetConfigLargeDirThreshold = "100MB"
	req.Args = []string{"machine", "backup"}
	return nil
}
```