# Scenario

**Feature**: fetching one Command Code account through the provider API

```
home (auth.json) + apiURL -> Service.fetch -> Response{status, plan, cycle, windows}
```

## Preconditions

1. Leaves serve the four read-only overlay endpoints from a fixture server.
2. Provider fetching is synchronous in tests (`TestExported_FetchOnce`).

## Steps

1. Set `Op` to the fetch outcome under test.

## Context

These are the outcomes the menu bar has to render: ready text, a missing
credential, an unreachable API, and a provider error response.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "ready"
	return nil
}
```
