# Scenario

**Feature**: PATCH unknown id returns 404

```
PATCH ?id=no_such -> 404
```

## Preconditions

1. Empty tree; id no_such.

## Steps

1. PATCH name X on missing id.
2. Assert 404.

## Context

Not found.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.APIOp = "patch"
	req.ID = "no_such"
	req.Name = "X"
	return nil
}
```
