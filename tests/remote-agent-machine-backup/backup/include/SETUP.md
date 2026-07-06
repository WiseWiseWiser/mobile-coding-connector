# Scenario

**Feature**: backup --include re-includes a built-in excluded path

```
# effective = (defaults - include) ∪ exclude
--include .cache -> .cache tree in DOT DIRS, not EXCLUDED
```

## Preconditions

Default `serverHome` fixtures with `.cache/junk`.

## Steps

1. `IncludePaths=[".cache"]`.
2. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/include`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.IncludePaths = []string{".cache"}
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```