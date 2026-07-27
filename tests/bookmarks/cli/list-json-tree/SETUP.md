# Scenario

**Feature**: list --json dumps document JSON

```
bookmarks list --json -> version and roots keys
```

## Preconditions

1. Optional seed one url for non-trivial tree.

## Steps

1. Seed bm_j; list --json.
2. Assert JSON fields in stdout.

## Context

Machine-readable list.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.SeedAdds = []SeedNode{{
		Type: "url", ID: "bm_j", Name: "JSON Item", URL: "https://json.example.com",
	}}
	req.CLIArgs = []string{"bookmarks", "list", "--json"}
	return nil
}
```
