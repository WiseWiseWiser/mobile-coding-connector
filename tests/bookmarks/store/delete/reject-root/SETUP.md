# Scenario

**Feature**: cannot delete fixed root folder

```
Delete id=root -> error; root still present
```

## Preconditions

1. ID root.

## Steps

1. Delete root.
2. Assert ErrMsg non-empty and root still in Doc.

## Context

Root is fixed.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ID = "root"
	return nil
}
```
