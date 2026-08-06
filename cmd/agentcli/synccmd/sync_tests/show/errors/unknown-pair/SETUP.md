# Scenario

**Feature**: show unknown pair name errors

```
operator -> show no-such -> error containing unknown pair
```

## Preconditions

- Empty store (or no matching pair).

## Steps

1. Args show no-such.
2. Assert error.

## Context

- Unknown pair.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"unison", "show", "no-such"}
	return nil
}
```
