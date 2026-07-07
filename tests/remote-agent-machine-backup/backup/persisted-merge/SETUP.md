# Scenario

**Feature**: persisted backup-config.json merges into dry-run plan

```
# prereq set-config, then dry-run without CLI --exclude
.knowledge-hub omitted from included paths; show-config shows from user config
```

## Preconditions

`serverHome` includes `.knowledge-hub` and `.knowledge-index` (`SeedKnowledgeHub`).

## Steps

1. `SeedKnowledgeHub=true`, `PrereqSetConfig=true`, `SetConfigExcludePaths=[".knowledge-hub", ".knowledge-index"]`.
2. Args: `machine backup --dry-run` (no CLI `--exclude`).
3. `FollowUpShowConfig=true` after dry-run.

## Context

Runtime merge of persisted user config with effective display reasons.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedKnowledgeHub = true
	req.PrereqSetConfig = true
	req.SetConfigExcludePaths = []string{".knowledge-hub", ".knowledge-index"}
	req.FollowUpShowConfig = true
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```