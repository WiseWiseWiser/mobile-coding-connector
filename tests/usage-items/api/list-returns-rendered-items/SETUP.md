# Scenario

**Feature**: GET /api/usage/items returns items with the menu text already rendered

```
GET /api/usage/items -> {"version":1,"default":"grok","rotate":true,"items":[{...,"title","dropdown"}]}
```

## Preconditions

1. The endpoint is public (auth skip list), because the menu-bar app reads it without a token.

## Steps

1. Set `Op=api`, `APIMethod=GET`, `APIPath=/api/usage/items`.

## Context

The app decodes this payload directly; no client-side formatting happens.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "api"
	req.APIMethod = "GET"
	req.APIPath = "/api/usage/items"
	return nil
}
```
