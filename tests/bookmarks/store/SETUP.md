# Scenario

**Feature**: bookmarks store persistence and tree mutations

```
# Manager at temp bookmarks.json
Load -> default root | Add/Update/Delete/Move -> Document on disk
```

## Preconditions

1. Mode `store`. Isolated temp store path from root Run.
2. Operations go through `bookmarks.Manager` only (no HTTP/CLI).

## Steps

1. Leaf sets `StoreOp` and node fields / PreAdds.
2. Run loads manager, applies op, returns DocView.
3. Assert checks tree shape and ErrMsg.

## Context

Core domain: Chrome-style folder/url tree, fixed root id=`root`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Mode = "store"
	return nil
}
```
