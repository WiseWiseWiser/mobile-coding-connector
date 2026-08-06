# Scenario

**Feature**: Status for a single named pair

```
Status(Name=mad-max) -> one StatusItem
```

## Preconditions

- Named status leaves.
- PairName set by leaves.

## Steps

1. Mode status.
2. Leaves seed pair and optional state.

## Context

- Covers never-run, seeded state, unknown pair.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "status"
	return nil
}
```
