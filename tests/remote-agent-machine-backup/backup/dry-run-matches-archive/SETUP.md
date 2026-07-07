# Scenario

**Feature**: dry-run plan included set matches streamed archive members

```
# CLI dry-run then CLI backup (same flags) -> plan.included == tar members - meta
```

## Preconditions

Default `serverHome` fixtures (built-in exclusions unchanged from dry-run walk).

## Steps

1. `DryRunThenArchive=true`.
2. `OutputPath=dry-run-matches-archive.tar.xz`.
3. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/dry-run-matches-archive`. `Run` executes dry-run, fetches
plan `included` via JSON API with matching exclude/include flags, then runs backup.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.DryRunThenArchive = true
	req.OutputPath = "dry-run-matches-archive.tar.xz"
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```