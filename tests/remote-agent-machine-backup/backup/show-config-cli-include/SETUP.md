# Scenario

**Feature**: backup --show-config merges CLI --include into effective preview

```
# prereq set-config excludes .cache; preview --include .cache removes it from effective exclude_paths
Merge(builtin, remote, CLI) -> .cache absent from exclude_paths
```

## Preconditions

Prereq `machine backup --set-config --exclude .cache` persists user exclude on server home.

## Steps

1. `PrereqSetConfig=true`, `SetConfigExcludePaths=[".cache"]`.
2. `ShowConfig=true`, `IncludePaths=[".cache"]`.
3. Args: `machine backup`.

## Context

REQUIREMENT leaf `backup/show-config-cli-include`. CLI `--include` overrides persisted and builtin exclude for preview.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.PrereqSetConfig = true
	req.SetConfigExcludePaths = []string{".cache"}
	req.ShowConfig = true
	req.IncludePaths = []string{".cache"}
	req.Args = []string{"machine", "backup"}
	return nil
}
```