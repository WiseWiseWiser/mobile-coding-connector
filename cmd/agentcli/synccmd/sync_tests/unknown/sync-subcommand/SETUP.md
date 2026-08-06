# Scenario

**Feature**: unknown top-level sync subcommand errors

```
operator -> RunCLI([foo]) -> unknown error
```

## Preconditions

- Args: `["foo"]` (not a known sync subcommand).

## Steps

1. Set Args to foo.
2. Assert error contains unknown.

## Context

- Sync dispatch.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"foo"}
	return nil
}
```
