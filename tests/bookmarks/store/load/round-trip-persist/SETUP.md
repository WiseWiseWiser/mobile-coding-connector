# Scenario

**Feature**: add url then reload Manager from disk preserves node

```
# Add url under root, new Manager.Load same path
id/name/url survive round-trip
```

## Preconditions

1. StoreOp add with fixed id; SecondOp reload.

## Steps

1. Add url `id=bm_dash` name Local Dashboard url http://127.0.0.1:7070.
2. Reload from path.
3. Assert node still present with same fields.

## Context

Durability of bookmarks.json.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.StoreOp = "add"
	req.NodeType = "url"
	req.ID = "bm_dash"
	req.Name = "Local Dashboard"
	req.URL = "http://127.0.0.1:7070"
	req.SecondOp = "reload"
	return nil
}
```
