# Scenario

**Feature**: HTTP bookmarks API on Manager via RegisterAPIWithManager

```
# httptest + Bearer token
GET/POST /api/bookmarks | PATCH/DELETE ?id= | POST /move
```

## Preconditions

1. Mode api; RegisterAPIWithManager mounted.
2. Auth wrapper requires Bearer token.

## Steps

1. Leaf sets APIOp and body/query.
2. Run performs request and snapshots GET tree into Doc.
3. Assert status and tree.

## Context

Server surface for CLI and menu bar clients.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Mode = "api"
	return nil
}
```
