# Scenario

**Feature**: `remote-agent machine backup` snapshots server HOME dot entries

```
# walk serverHome dot children, apply exclusions, dry-run plan or tar.xz stream
remote-agent machine backup [flags] -> JSON plan | tar.xz archive
```

## Preconditions

`serverHome` seeded with included dot paths and built-in exclusion trees.

## Steps

1. Leaf sets `Request.Args`, `OutputPath`, and `ExcludePaths`.
2. `Run` executes backup against live server.
3. `Assert` checks stdout plan or output archive members.

## Context

Grouping node for backup: dry-run plan, streamed archive, and custom `--exclude`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if len(req.Args) >= 2 && req.Args[0] == "machine" && req.Args[1] != "backup" {
		t.Fatalf("backup group: unexpected subcommand argv %v", req.Args)
	}
	return nil
}
```