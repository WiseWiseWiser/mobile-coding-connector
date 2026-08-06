# Scenario

**Feature**: rm unknown pair errors

```
operator -> rm no-such -> unknown pair
```

## Preconditions

- Empty store.

## Steps

1. Rm unknown.
2. Assert error.

## Context

- Unknown rm.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"unison", "rm", "no-such", "--yes"}
	return nil
}
```
