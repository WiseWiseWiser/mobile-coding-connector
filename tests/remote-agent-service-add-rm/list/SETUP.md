# Scenario

**Feature**: `remote-agent service list` group (plain list + `--all` alias)

```
service list [--all]
```

## Preconditions

1. Leaves seed two services.
2. Both plain `list` and `list --all` must show every service.

## Steps

1. Leaves seed rows and choose flags.
2. Root Run executes `agentcli.Run` and snapshots List + on-disk JSON.

## Context

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// Group default: service list family (plain or --all).
	// Leaves seed rows and choose flags.
	return nil
}
```
