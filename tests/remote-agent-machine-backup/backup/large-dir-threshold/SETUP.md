# Scenario

**Feature**: --large-dir-threshold raises flag cutoff above seeded dir size

```
# .big-test/ is 50MB; --large-dir-threshold 100MB -> no LARGE SIZE, no detail
```

## Preconditions

`SeedLargeDir` writes `.big-test/` totaling ~50 MB.

## Steps

1. `SeedLargeDir=true`.
2. Args: `machine backup --dry-run --large-dir-threshold 100MB`.

## Context

REQUIREMENT leaf `backup/large-dir-threshold`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedLargeDir = true
	req.Args = []string{"machine", "backup", "--dry-run", "--large-dir-threshold", "100MB"}
	return nil
}
```