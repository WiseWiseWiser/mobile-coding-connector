# Scenario

**Feature**: POST /api/bookmarks/move reparents node

```
POST move {id,parent_id} -> child under new parent
```

## Preconditions

1. Seeds folder + url; APIOp move.

## Steps

1. Move bm_mv under fld_api.
2. Assert reparent.

## Context

Move endpoint.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.SeedAdds = []SeedNode{
		{Type: "folder", ID: "fld_api", Name: "F"},
		{Type: "url", ID: "bm_mv", Name: "MoveMe", URL: "https://mv.example.com"},
	}
	req.APIOp = "move"
	req.ID = "bm_mv"
	req.MoveParentID = "fld_api"
	return nil
}
```
