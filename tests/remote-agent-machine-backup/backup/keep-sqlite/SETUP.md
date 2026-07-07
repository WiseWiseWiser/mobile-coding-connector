# Scenario

**Feature**: backup keeps SQLite database files (not treated as executable binary)

```
# opencode.db has SQLite magic -> included in DOT FILES and archive
```

## Preconditions

`serverHome` includes `.local/share/opencode/opencode.db` with SQLite header bytes.

## Steps

1. Set `OutputPath` under `agentHome`.
2. Args: `machine backup --output <path>`.

## Context

REQUIREMENT leaf `backup/keep-sqlite`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.OutputPath = "keep-sqlite-backup.tar.xz"
	req.Args = []string{"machine", "backup", "--output", "__OUTPUT_PATH__"}
	return nil
}
```