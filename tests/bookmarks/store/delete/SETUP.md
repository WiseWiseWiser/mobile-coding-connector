# Scenario

**Feature**: delete url or folder (recursive); protect fixed root

```
# Manager.Delete(id)
url removed; folder removes descendants; root rejected
```

## Preconditions

1. StoreOp delete; seeds as needed.

## Steps

1. Delete target id.
2. Assert absence or error for root.

## Context

Tree pruning rules.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.StoreOp = "delete"
	return nil
}
```
