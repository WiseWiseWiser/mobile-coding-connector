# Scenario

**Feature**: backup --dry-run prints plan without writing an archive

```
# server walk + exclusions -> SSE /backup/stream -> two-phase stdout
stream: DOT FILES/DIRS/EXCLUDED with sizes; summary: dry-run: machine backup plan
```

## Preconditions

Default `serverHome` fixtures.

## Steps

1. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/dry-run`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```