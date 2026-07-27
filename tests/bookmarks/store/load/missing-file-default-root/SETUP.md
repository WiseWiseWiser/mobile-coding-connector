# Scenario

**Feature**: missing bookmarks.json yields default empty root folder

```
# no file on disk
Load -> {version:1, roots:[{id:root, type:folder, name:Bookmarks, children:[]}]}
```

## Preconditions

1. No SeedJSON; store path does not exist before Load.

## Steps

1. StoreOp `load`.
2. Assert version, single root id/name/type, empty children.

## Context

Requirement: missing file → default root.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.StoreOp = "load"
	req.SeedJSON = ""
	return nil
}
```
