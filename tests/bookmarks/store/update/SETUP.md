# Scenario

**Feature**: update name, url, and browser on existing nodes

```
# Manager.Update(id, opts)
name/url/browser change; ClearBrowser clears optional browser
```

## Preconditions

1. PreAdds seed node; StoreOp update.

## Steps

1. Leaf seeds id then updates fields.
2. Assert updated node fields.

## Context

PATCH-equivalent domain ops.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.StoreOp = "update"
	return nil
}
```
