# Scenario

**Feature**: restore --dry-run reports update for changed files

```
# backup, mutate .bashrc, restore --dry-run via /restore/stream
stream update: for .bashrc + skip lines; summary dry-run: machine restore plan
```

## Preconditions

Prereq backup; `.bashrc` mutated after backup.

## Steps

1. `AfterBackupMutate=modify-bashrc`.
2. Args: `machine restore --dry-run`.

## Context

REQUIREMENT leaf `restore/dry-run-changed`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.AfterBackupMutate = "modify-bashrc"
	req.Args = []string{"machine", "restore", "--dry-run"}
	return nil
}
```