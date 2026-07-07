# Scenario

**Feature**: backup --set-config persists user excludes to backup-config.json

```
# set-config writes ~/.ai-critic/backup-config.json on server
remote-agent machine backup --set-config --exclude .knowledge-hub -> persisted JSON
```

## Preconditions

`serverHome` includes `.knowledge-hub` fixtures (`SeedKnowledgeHub`).

## Steps

1. `SeedKnowledgeHub=true`, `SetConfig=true`, `ExcludePaths=[".knowledge-hub"]`.
2. Args: `machine backup`.

## Context

Persisted backup config leaf.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedKnowledgeHub = true
	req.SetConfig = true
	req.ExcludePaths = []string{".knowledge-hub"}
	req.Args = []string{"machine", "backup"}
	return nil
}
```