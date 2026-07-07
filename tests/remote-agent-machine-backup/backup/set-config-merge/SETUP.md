# Scenario

**Feature**: incremental --set-config --exclude merges into persisted exclude_paths

```
# prereq set-config .knowledge-hub, then set-config --exclude .docker only
persisted backup-config.json contains both paths
```

## Preconditions

`serverHome` includes `.knowledge-hub` fixtures (`SeedKnowledgeHub`).

## Steps

1. `SeedKnowledgeHub=true`, `PrereqSetConfig=true`, `SetConfigExcludePaths=[".knowledge-hub"]`.
2. Main invocation: `SetConfig=true`, `ExcludePaths=[".docker"]`.
3. Args: `machine backup`.

## Context

Set-config merge leaf: second exclude-only invocation must union with prior persisted paths.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedKnowledgeHub = true
	req.PrereqSetConfig = true
	req.SetConfigExcludePaths = []string{".knowledge-hub"}
	req.SetConfig = true
	req.ExcludePaths = []string{".docker"}
	req.Args = []string{"machine", "backup"}
	return nil
}
```