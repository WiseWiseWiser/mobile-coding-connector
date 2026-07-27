# Scenario

**Feature**: bookmarks move reparents under folder

```
bookmarks move bm_cm --parent fld_c
```

## Preconditions

1. Seed folder + url.

## Steps

1. move.
2. Assert under folder.

## Context

CLI move.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.SeedAdds = []SeedNode{
		{Type: "folder", ID: "fld_c", Name: "CF"},
		{Type: "url", ID: "bm_cm", Name: "CMove", URL: "https://cm.example.com"},
	}
	req.CLIArgs = []string{"bookmarks", "move", "bm_cm", "--parent", "fld_c"}
	return nil
}
```
