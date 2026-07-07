# Scenario

**Feature**: backup --include re-includes a specific executable excluded by binary rule

```
# includedPaths exact match overrides **(binary) -> stub in DOT FILES
```

## Preconditions

`serverHome` includes ELF stub at `.ai-critic/bin/stub`.

## Steps

1. `IncludePaths=[".ai-critic/bin/stub"]`.
2. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/include-binary`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.IncludePaths = []string{".ai-critic/bin/stub"}
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```