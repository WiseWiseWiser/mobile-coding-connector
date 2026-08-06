# Scenario

**Feature**: unison help advertises doctor and status

```
operator -> RunCLI([unison --help]) -> Usage listing doctor + status
```

## Preconditions

- Grouping node for help family.
- Mode will be `cli`.

## Steps

1. Set Mode to `cli` when empty.
2. Leaves set concrete Args.

## Context

- L2 in-process CLI only.

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
	if req.Args == nil {
		req.Args = []string{}
	}
	return nil
}
```
