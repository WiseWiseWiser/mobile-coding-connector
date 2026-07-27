# Scenario

**Feature**: List vs ListAll project scoping on services.Manager

```
seed multi-project definitions -> Manager.List / ListAll -> scoped or all IDs
```

## Preconditions

1. Two service definitions with different `projectDir` values.
2. L2 harness uses `services.NewManagerFromDefinitions` (no product binary).

## Steps

1. Root `Run` builds in-memory Manager with local + other project services.
2. Leaf `Setup` sets `Op` to `list-scoped` or `list-all`.
3. Root `Run` calls `List(projectDir)` or `ListAll()`.
4. Leaf `Assert` checks returned service IDs.

## Context

Menu bar uses `?all=1` (ListAll) to show every managed service.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	return nil
}
```
