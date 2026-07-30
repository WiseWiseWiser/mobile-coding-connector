# Scenario

**Feature**: `remote-agent service rm` group

```
service rm <name-or-id>
```

## Preconditions

1. Leaves seed definitions via `Request.Services` when a target must exist.
2. Resolution must use list-all (cross-projectDir) once implemented.

## Steps

1. Leaf seeds and sets CLIArgs for rm.
2. Run executes agentcli against L2 mux.
3. Assert Removed / errors and ListAll membership.

## Context

Subcommands use exact verb `rm` only (no `remove` / `delete` aliases).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// Group default: service rm family (exact verb "rm"; no remove/delete aliases).
	// Leaves seed Services and set name-or-id argv when needed.
	if len(req.CLIArgs) == 0 {
		req.CLIArgs = []string{"service", "rm"}
	}
	return nil
}
```

