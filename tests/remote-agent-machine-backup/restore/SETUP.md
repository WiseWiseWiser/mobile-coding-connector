# Scenario

**Feature**: `remote-agent machine restore` applies tar.xz archives to server HOME

```
# prereq backup -> optional mutate serverHome -> restore --dry-run|apply
skip identical paths; create/update changed paths
```

## Preconditions

`serverHome` seeded; `Run` creates a prereq backup when `PrereqBackup` is true.

## Steps

1. Leaf sets `PrereqBackup`, `AfterBackupMutate`, and restore `Args`.
2. `Run` backs up, mutates server home if requested, then runs restore.
3. `Assert` checks skip lines, plan actions, and on-disk file contents.

## Context

Grouping node for restore: identical skips, changed dry-run plan, and apply.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if len(req.Args) >= 2 && req.Args[0] == "machine" && req.Args[1] != "restore" {
		t.Fatalf("restore group: unexpected subcommand argv %v", req.Args)
	}
	req.PrereqBackup = true
	return nil
}
```