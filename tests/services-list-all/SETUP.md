# Scenario

**Feature**: List / ListAll return every managed service

```
seed two definitions -> Manager.List / ListAll -> both IDs
```

## Preconditions

1. Two service definitions in memory.
2. L2 harness uses `services.NewManagerFromDefinitions` (no product binary).

## Steps

1. Root `Run` builds in-memory Manager with two services.
2. Leaf `Setup` sets `Op` to `list` or `list-all`.
3. Root `Run` calls `List()` or `ListAll()`.
4. Leaf `Assert` checks returned service IDs.

## Context

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	return nil
}
```
