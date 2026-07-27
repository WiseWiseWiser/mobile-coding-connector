# Scenario

**Feature**: add folder or url nodes under a parent

```
# Manager.Add(parent, node)
tree gains child under parent (default parent root)
```

## Preconditions

1. StoreOp `add`.

## Steps

1. Leaf sets type/name/url/id.
2. Assert child under root (or custom parent).

## Context

Create path for tree growth.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.StoreOp = "add"
	return nil
}
```
