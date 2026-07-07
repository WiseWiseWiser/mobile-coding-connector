# Scenario

**Feature**: dry-run GIT REPOS reports none when no git repos discovered

```
# default serverHome without git fixtures -> backup --dry-run -> GIT REPOS: (none)
```

## Preconditions

Default `serverHome` fixtures (no git repos under included dot-dirs).

## Steps

1. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/git-repos-none`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```