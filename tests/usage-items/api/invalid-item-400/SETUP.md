# Scenario

**Feature**: a Command Code item without a home is rejected as invalid

```
POST /api/usage/items/add {"id":"cc","kind":"commandcode"} -> 400 --home is required
```

## Preconditions

1. The registry is seeded.

## Steps

1. Set `Op=api` and POST an add with no home.

## Context

Both the CLI and the server reject the same way, so neither can register an unusable item.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "api"
	req.APIMethod = "POST"
	req.APIPath = "/api/usage/items/add"
	req.APIBody = `{"id":"cc","kind":"commandcode"}`
	return nil
}
```
