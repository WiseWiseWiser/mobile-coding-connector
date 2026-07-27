# Scenario

**Feature**: bookmarks delete removes id from tree

```
bookmarks delete bm_del
```

## Preconditions

1. Seed bm_del.

## Steps

1. delete.
2. Assert absent.

## Context

CLI delete.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.SeedAdds = []SeedNode{{
		Type: "url", ID: "bm_del", Name: "DeleteMe", URL: "https://del.example.com",
	}}
	req.CLIArgs = []string{"bookmarks", "delete", "bm_del"}
	return nil
}
```
