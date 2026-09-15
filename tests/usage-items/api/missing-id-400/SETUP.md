# Scenario

**Feature**: a remove without an id is rejected before touching the registry

```
POST /api/usage/items/remove {} -> 400 usage item: --id is required
```

## Preconditions

1. The registry is seeded.

## Steps

1. Set `Op=api` and POST a remove with an empty body.

## Context

An accidental empty remove must not fall through to a not-found error.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "api"
	req.APIMethod = "POST"
	req.APIPath = "/api/usage/items/remove"
	req.APIBody = `{}`
	return nil
}
```
