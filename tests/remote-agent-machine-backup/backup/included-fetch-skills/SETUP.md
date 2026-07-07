# Scenario

**Feature**: reverted built-in exclusions leave fetch-skill and knowledge-index paths included

```
# seed git-fetch / confluence-fetch / knowledge-index -> dry-run plan includes paths
```

## Preconditions

`SeedIncludedFetchSkills` writes small files under the three removed exclusion paths.

## Steps

1. `SeedIncludedFetchSkills=true`.
2. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/included-fetch-skills`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedIncludedFetchSkills = true
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```