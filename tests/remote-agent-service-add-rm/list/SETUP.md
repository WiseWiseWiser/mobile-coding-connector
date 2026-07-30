# Scenario

**Feature**: `remote-agent service list` group (`--all` + scoped)

```
service list [--project-dir DIR] [--all]
```

## Preconditions

1. Leaves seed two services with distinct absolute `projectDir` values.
2. `list --all` must call `GET /api/services?all=1` once implemented.

## Steps

1. Leaf seeds multi-scope services and sets CLIArgs.
2. Run executes agentcli against L2 mux.
3. Assert which names appear in stdout / ListAll snapshot.

## Context

Scoped list without `--all` still uses `--project-dir` for deterministic L2
filtering (avoids depending on process cwd as default scope).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// Group default: service list family (--all or --project-dir).
	// Leaves seed multi-projectDir rows and choose flags.
	if len(req.CLIArgs) == 0 {
		req.CLIArgs = []string{"service", "list"}
	}
	return nil
}
```

