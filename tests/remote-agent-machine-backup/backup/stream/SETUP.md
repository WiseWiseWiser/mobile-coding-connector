# Scenario

**Feature**: backup streams a tar.xz archive with manifest and included members

```
# server walk -> tar.xz on disk
archive contains manifest.json + dot paths; exclusions omitted
```

## Preconditions

Default `serverHome` fixtures.

## Steps

1. Set `OutputPath` under `agentHome`.
2. Args: `machine backup --output <path>`.

## Context

REQUIREMENT leaf `backup/stream`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.OutputPath = "stream-backup.tar.xz"
	req.Args = []string{"machine", "backup", "--output", "__OUTPUT_PATH__"}
	return nil
}
```