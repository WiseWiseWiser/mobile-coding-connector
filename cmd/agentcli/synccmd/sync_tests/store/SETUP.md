# Scenario

**Feature**: pairs.json edge cases

```
corrupt store -> CLI surfaces error
```

## Preconditions

- Grouping node for this operation family (not a runnable leaf).
- Injectable dirs already set by root Setup.

## Steps

1. Normalize `Args` to non-nil empty when unset so leaves only specialize.
2. Leaves override `Args` / `PreArgs` / `FocusPair`.

## Context

- Inherited root contract; L2 in-process.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	if req.Args == nil {
		req.Args = []string{}
	}
	return nil
}
```
