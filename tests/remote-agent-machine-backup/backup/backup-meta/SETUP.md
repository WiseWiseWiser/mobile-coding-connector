# Scenario

**Feature**: real backup archive contains phantom .backup/ meta entries

```
# pack-time injection of config.json, installed.json, ENV
seeded ~/.backup/config.json -> .backup/config.json.machine.bak in archive
```

## Preconditions

`SeedBackupMeta=true` seeds distinguishable old JSON at `serverHome/.backup/config.json`.

## Steps

1. `SeedBackupMeta=true`, `OutputPath=backup-meta.tar.xz`.
2. Args: `machine backup --output <path>`.

## Context

REQUIREMENT leaf `backup/backup-meta`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedBackupMeta = true
	req.OutputPath = "backup-meta.tar.xz"
	req.Args = []string{"machine", "backup", "--output", "__OUTPUT_PATH__"}
	return nil
}
```