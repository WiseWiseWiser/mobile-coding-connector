# Scenario

**Feature**: CLI `unison run` dispatch via RunCLI

```
operator -> RunCLI([unison run <name> flags], CLIOpts{Exec, probes})
  -> state + writers
```

## Preconditions

- Grouping for CLI run leaves.
- CLIOpts.Exec injected by harness.

## Steps

1. Default Mode to `cli` when empty.
2. Leaves set Args and seeds.

## Context

- Wire path for operator-facing run.

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
