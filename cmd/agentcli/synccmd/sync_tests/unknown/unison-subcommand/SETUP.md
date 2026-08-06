# Scenario

**Feature**: unknown unison subcommand errors

```
operator -> RunCLI([unison, frobnicate]) -> unknown error
```

## Preconditions

- Args: `unison frobnicate`.

## Steps

1. Set Args.
2. Assert unknown.

## Context

- Unison dispatch.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"unison", "frobnicate"}
	return nil
}
```
