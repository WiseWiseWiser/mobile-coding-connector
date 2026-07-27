# Scenario

**Feature**: store rejects invalid node fields

```
# Add with empty name or bad url
error; tree unchanged for invalid node
```

## Preconditions

1. StoreOp add with invalid fields.

## Steps

1. Attempt add.
2. Assert ErrMsg and absence of invalid node.

## Context

Domain validation.

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
