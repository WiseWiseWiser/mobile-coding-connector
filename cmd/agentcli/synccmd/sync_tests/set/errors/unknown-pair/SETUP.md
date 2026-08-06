# Scenario

**Feature**: set unknown pair errors

```
operator -> set no-such --prefer newer -> unknown pair
```

## Preconditions

- Empty store.

## Steps

1. Args set unknown.
2. Assert error.

## Context

- Unknown pair on set.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"unison", "set", "no-such", "--prefer", "newer"}
	return nil
}
```
