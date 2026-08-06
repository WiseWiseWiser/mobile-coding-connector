# Scenario

**Feature**: empty args print sync help

```
operator -> RunCLI([]) -> sync Usage on stdout, trailing \\n
```

## Preconditions

- Args: empty slice (after `sync`).

## Steps

1. Set Args to empty.
2. RunCLI; Assert Usage.

## Context

- Sync top-level help.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{}
	return nil
}
```
