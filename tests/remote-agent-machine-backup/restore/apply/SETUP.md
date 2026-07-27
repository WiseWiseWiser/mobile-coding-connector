# Scenario

**Feature**: restore apply writes changed files and skips identical paths

```
# backup, mutate .bashrc, restore apply via /restore/stream?dry_run=false
CLASSIFYING: all entries; APPLYING: update .bashrc; machine restore summary
```

## Preconditions

Prereq backup; `.bashrc` mutated after backup.

## Steps

1. `AfterBackupMutate=modify-bashrc`.
2. Args: `machine restore` (no `--dry-run`).

## Context

REQUIREMENT leaf `restore/apply`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// L3 smoke: restore apply via product binaries.
	req.UseCLI = true
	req.AfterBackupMutate = "modify-bashrc"
	req.Args = []string{"machine", "restore"}
	return nil
}
```