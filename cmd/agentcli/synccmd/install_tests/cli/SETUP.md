# Scenario

**Feature**: CLI unison install with injectable ensure hooks

```
operator -> RunCLI([unison install …], CLIOpts{LocalEnsure, RemoteEnsure})
  -> hooks + RunErr
```

## Preconditions

- Grouping for CLI install leaves.
- Leaves set Args; harness injects hooks on CLIOpts.

## Steps

1. Default Mode to `cli` when empty.
2. Leaves set Args (default-both: no scope flags).

## Context

- CLI wire for P4; library scopes covered under `library/`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	if req.Mode == "" {
		req.Mode = "cli"
	}
	return nil
}
```
