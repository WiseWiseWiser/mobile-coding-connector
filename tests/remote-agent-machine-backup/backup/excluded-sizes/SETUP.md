# Scenario

**Feature**: EXCLUDED section reports per-rule FILES and SIZE totals sorted by size

```
# walk attributes skipped bytes to first matching rule -> one progress line per rule
machine backup --dry-run -> EXCLUDED table (RULE FILES SIZE REASON) + summary totals
```

## Preconditions

`serverHome` seeded with known-size fixtures: `.cache/junk` (1024 B), `.cache/nested/deep`
(512 B), `.ai-critic/service.log` (512 B), plus default exclusion trees (`.npm`, etc.).

## Steps

1. Set `SeedExcludedSizes` so `Run` writes deterministic byte sizes.
2. Args: `machine backup --dry-run`.

## Context

REQUIREMENT-DESIGN-excluded-sizes.md leaf `backup/excluded-sizes`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedExcludedSizes = true
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```