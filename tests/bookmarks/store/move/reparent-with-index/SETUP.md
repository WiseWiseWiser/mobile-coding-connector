# Scenario

**Feature**: move url into folder at index 0

```
Move bm_m -> parent fld_m index 0
```

## Preconditions

1. Seeds folder fld_m and url bm_m under root.

## Steps

1. Move bm_m to fld_m index 0.
2. Assert bm_m is child of fld_m; not direct root child.

## Context

Move API domain.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.PreAdds = []SeedNode{
		{Type: "folder", ID: "fld_m", Name: "M"},
		{Type: "url", ID: "bm_m", Name: "Movable", URL: "https://m.example.com"},
	}
	req.ID = "bm_m"
	req.MoveParentID = "fld_m"
	idx := 0
	req.Index = &idx
	return nil
}
```
