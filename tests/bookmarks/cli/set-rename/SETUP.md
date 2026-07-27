# Scenario

**Feature**: bookmarks set renames node

```
bookmarks set bm_set --name RenamedCLI
```

## Preconditions

1. Seed bm_set.

## Steps

1. set --name.
2. Assert Doc name.

## Context

CLI update.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.SeedAdds = []SeedNode{{
		Type: "url", ID: "bm_set", Name: "Before", URL: "https://set.example.com",
	}}
	req.CLIArgs = []string{"bookmarks", "set", "bm_set", "--name", "RenamedCLI"}
	return nil
}
```
