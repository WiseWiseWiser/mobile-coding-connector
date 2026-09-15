# Scenario

**Feature**: an injected fetch error becomes the item's error text

```
fetcher returns an error -> status error with that message
```

## Preconditions

1. The fetcher is injected and returns `token rejected by provider`.

## Steps

1. Set `Op=injected-error`.

## Context

Provider-specific messages (expired session, rejected token) must reach the menu bar
unchanged so the user knows what to fix.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "injected-error"
	return nil
}
```
