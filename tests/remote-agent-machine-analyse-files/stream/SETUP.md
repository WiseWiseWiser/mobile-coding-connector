# Scenario

**Feature**: `remote-agent machine analyse-files` streams per-entry HOME scan blocks

```
# walk serverHome children -> SSE /analyse-files/stream -> entry blocks + summary
remote-agent machine analyse-files -> home line + > blocks + analyse-files summary
```

## Preconditions

`serverHome` seeded per leaf `SeedProfile`.

## Steps

1. Leaf sets `Request.SeedProfile` (and optional `Args`).
2. `Run` executes analyse-files against live server.
3. `Assert` checks stdout entry blocks and summary.

## Context

Grouping node for streamed analyse-files scenarios from REQUIREMENT-DESIGN-machine-analyse-files.md.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if len(req.Args) >= 2 && req.Args[0] == "machine" && req.Args[1] != "analyse-files" {
		t.Fatalf("stream group: unexpected subcommand argv %v", req.Args)
	}
	return nil
}
```