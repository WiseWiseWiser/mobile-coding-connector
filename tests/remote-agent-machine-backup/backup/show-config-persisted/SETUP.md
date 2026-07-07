# Scenario

**Feature**: show-config displays persisted user excludes with display reasons

```
# prereq set-config for .knowledge-hub, patch manual reason for .knowledge-index, then show-config
CLI-set path -> from user config; hand-edited reason preserved
```

## Preconditions

`serverHome` includes `.knowledge-hub` and `.knowledge-index` (`SeedKnowledgeHub`).

## Steps

1. `SeedKnowledgeHub=true`, `PrereqSetConfig=true`, `SetConfigExcludePaths=[".knowledge-hub"]`.
2. `PostPrereqSetConfigExcludes` adds `.knowledge-index` with reason `knowledge index cache`.
3. `ShowConfig=true`, Args: `machine backup`.

## Context

Effective merged display reasons for user-authored backup-config.json.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedKnowledgeHub = true
	req.PrereqSetConfig = true
	req.SetConfigExcludePaths = []string{".knowledge-hub"}
	req.PostPrereqSetConfigExcludes = []PostSetConfigExcludeEntry{
		{Path: ".knowledge-index", Reason: "knowledge index cache"},
	}
	req.ShowConfig = true
	req.Args = []string{"machine", "backup"}
	return nil
}
```