# Scenario

**Feature**: usage item HTTP endpoints behind the real mux

```
GET  /api/usage/items          -> rendered items + selection (public)
POST /api/usage/items/{add,update,remove,default}
     -> 200 | 400 invalid | 404 not_found | 409 conflict
```

## Preconditions

1. `server/usage.RegisterAPI` serves the handlers; `TestExported_SetItemsService`
   points them at a service over a temporary registry.
2. `GET /api/usage/items` is on the server's auth skip list, so the menu-bar app
   can read it without a token.

## Steps

1. Set `Op=api` plus `APIMethod`, `APIPath`, and `APIBody`.

## Context

Status mapping is what the CLI turns into `warning:` / `Error:` output, so the
codes are part of the user-visible contract.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "api"
	return nil
}
```
