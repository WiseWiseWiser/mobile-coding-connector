# Scenario

**Feature**: adding an id that already exists is a conflict

```
POST /api/usage/items/add {"id":"grok"} -> 409 usage item "grok" already exists
```

## Preconditions

1. The registry is seeded, so `grok` is taken.

## Steps

1. Set `Op=api` and POST an add for the existing id.

## Context

The CLI turns 409 into a plain `Error:` line pointing at `update`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "api"
	req.APIMethod = "POST"
	req.APIPath = "/api/usage/items/add"
	req.APIBody = `{"id":"grok","kind":"grok"}`
	return nil
}
```
