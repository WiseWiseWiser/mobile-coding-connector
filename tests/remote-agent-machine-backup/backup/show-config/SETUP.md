# Scenario

**Feature**: backup --show-config prints effective merged exclusion config JSON

```
# server effective config (builtin + persisted backup-config.json)
remote-agent machine backup --show-config -> JSON version + exclude_paths
```

## Preconditions

Default `serverHome` fixtures; server serves GET /api/remote-agent/machine/backup-config.

## Steps

1. `ShowConfig=true`.
2. Args: `machine backup`.

## Context

REQUIREMENT leaf `backup/show-config`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ShowConfig = true
	req.Args = []string{"machine", "backup"}
	return nil
}
```