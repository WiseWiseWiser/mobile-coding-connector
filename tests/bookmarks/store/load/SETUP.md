# Scenario

**Feature**: load bookmarks.json (missing file / round-trip)

```
# missing file or after write
Manager.Load -> Document version=1 root Bookmarks
```

## Preconditions

1. StoreOp load or add+reload under Mode store (set by parent).

## Steps

1. Leaf configures SeedJSON / add + SecondOp reload.
2. This grouping ensures StoreOp defaults toward load when leaf omits.
3. Assert default root or persisted node.

## Context

Empty state and durability.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// Group default: pure load unless leaf overrides StoreOp.
	if req.StoreOp == "" {
		req.StoreOp = "load"
	}
	return nil
}
```
