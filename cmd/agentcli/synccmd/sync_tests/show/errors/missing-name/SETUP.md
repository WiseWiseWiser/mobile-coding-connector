# Scenario

**Feature**: show without name errors

```
operator -> show (no name) -> error
```

## Preconditions

- Args: `unison show` only.

## Steps

1. Set Args without name.
2. Assert non-empty error.

## Context

- Missing positional.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"unison", "show"}
	return nil
}
```
