# Scenario

**Feature**: backup --include re-includes a specific .log file excluded by suffix rule

```
# includedPaths exact match overrides **/*.log -> keep.log in DOT FILES
```

## Preconditions

`serverHome` includes `.ai-critic/keep.log` and `.ai-critic/service.log`.

## Steps

1. `IncludePaths=[".ai-critic/keep.log"]`.
2. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/include-log`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.IncludePaths = []string{".ai-critic/keep.log"}
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```