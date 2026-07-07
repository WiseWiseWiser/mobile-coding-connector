# Scenario

**Feature**: backup archive omits new built-in path-prefix exclusion trees

```
# path prefix rules exclude seeded trees from tar.xz members
```

## Preconditions

`serverHome` includes trees under `.codex/.tmp`, `.local/share/opencode/repos`,
`.local/share/cursor-agent/versions`, `.opencode/bin`, and
`.config/confluence-fetch-skill/data` (now included, not excluded).

## Steps

1. Set `OutputPath` under `agentHome`.
2. Args: `machine backup --output <path>`.

## Context

REQUIREMENT leaf `backup/path-exclusions`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.OutputPath = "path-exclusions-backup.tar.xz"
	req.Args = []string{"machine", "backup", "--output", "__OUTPUT_PATH__"}
	return nil
}
```