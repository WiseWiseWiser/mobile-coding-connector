# Scenario

**Feature**: delete folder removes all descendants

```
Delete fld_dev containing child url -> both gone
```

## Preconditions

1. Folder seed + child under folder (two-step PreAdds with Parent).

## Steps

1. Seed folder fld_dev and url bm_inner parent fld_dev.
2. Delete fld_dev.
3. Assert both missing.

## Context

Recursive folder delete.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.PreAdds = []SeedNode{
		{Type: "folder", ID: "fld_dev", Name: "Dev"},
		{Type: "url", ID: "bm_inner", Name: "Inner", URL: "https://inner.example.com", Parent: "fld_dev"},
	}
	req.ID = "fld_dev"
	return nil
}
```
