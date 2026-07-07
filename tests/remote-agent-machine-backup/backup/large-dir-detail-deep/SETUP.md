# Scenario

**Feature**: dry-run summary deep-scans included dirs for flat LARGE DIR DETAIL

```
# seed .deep-test/nested-big/ 12MB + excluded .cache -> machine backup --dry-run
summary flat detail lists nested path; builtin-excluded .cache absent
```

## Preconditions

`SeedLargeDirDetailDeep` writes `.big-test/` (50 MB), `.deep-test/nested-big/file` (12 MB),
`.deep-test/small/tiny` (1 KB), and reuses default `.cache` (builtin excluded).

## Steps

1. `SeedLargeDirDetailDeep=true`.
2. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/large-dir-detail-deep`. Detail threshold is fixed 10 MB;
`LARGE SIZE` still uses default 40 MB.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedLargeDirDetailDeep = true
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```