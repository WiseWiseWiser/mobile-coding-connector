# Scenario

**Feature**: `remote-agent service add` group

```
service add --name … --command … [options]
```

## Preconditions

1. Empty or leaf-specific `services.json` under isolated config home.
2. Server accepts POST create (existing Manager API).

## Steps

1. Leaf sets CLIArgs for add (or invalid flag combinations).
2. Run executes agentcli against L2 mux.
3. Assert exit, stdout, disk, and ListAll.

## Context

Subcommands use exact verb `add` only (no `create` alias).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// Group default: service add family (exact verb "add"; no create alias).
	// Leaves replace CLIArgs fully; no pre-seeded definitions at group level.
	req.Services = nil
	if len(req.CLIArgs) == 0 {
		req.CLIArgs = []string{"service", "add"}
	}
	return nil
}
```

