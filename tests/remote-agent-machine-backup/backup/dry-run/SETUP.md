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
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// L3 smoke: backup --dry-run plan via product binaries.
	req.UseCLI = true
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```