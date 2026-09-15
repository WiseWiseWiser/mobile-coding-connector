# Scenario

**Feature**: changing an unknown item is a 404

```
POST /api/usage/items/update {"id":"missing"} -> 404 unknown usage item "missing"
```

## Preconditions

1. The registry only holds grok and codex.

## Steps

1. Set `Op=api` and POST an update for an id that does not exist.

## Context

The CLI reports this as an error and suggests nothing else.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "api"
	req.APIMethod = "POST"
	req.APIPath = "/api/usage/items/update"
	req.APIBody = `{"id":"missing","label":"x"}`
	return nil
}
```
