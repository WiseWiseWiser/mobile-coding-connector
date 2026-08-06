# Scenario

**Feature**: set without name errors

```
operator -> set (no name) -> error
```

## Preconditions

- Args: `unison set` only.

## Steps

1. Set Args without name.
2. Assert error.

## Context

- Missing name.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"unison", "set"}
	return nil
}
```
