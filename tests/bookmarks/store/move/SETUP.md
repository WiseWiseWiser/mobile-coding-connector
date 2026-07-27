# Scenario

**Feature**: reparent nodes with optional index

```
# Manager.Move(id, parentID, index)
url moves under folder at index
```

## Preconditions

1. StoreOp move.

## Steps

1. Seed folder + url under root; move url into folder.
2. Assert new parent children.

## Context

Reorder / reparent.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.StoreOp = "move"
	return nil
}
```
