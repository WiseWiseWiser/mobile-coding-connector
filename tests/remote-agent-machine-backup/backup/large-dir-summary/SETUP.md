# Scenario

**Feature**: dry-run summary flags large included dirs and prints LARGE DIR DETAIL

```
# seed .big-test/ >40MB -> machine backup --dry-run -> LARGE SIZE + detail block
summary DOT DIRS sorted by size desc; plain LARGE SIZE (stdout piped, not TTY)
```

## Preconditions

`SeedLargeDir` writes `.big-test/` (50 MB) and `.small-test/` for sort contrast.

## Steps

1. `SeedLargeDir=true`.
2. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/large-dir-summary`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedLargeDir = true
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```