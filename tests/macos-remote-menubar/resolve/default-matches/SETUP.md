# Scenario

**Feature**: default matches a domain entry

```
default=https://example.com + domain token secret -> Resolve -> that server+token, state=ok
```

## Preconditions

Default string equals a domain `server` field (exact match after normalize).

## Steps

1. Set ConfigJSON with one matching domain.

## Context

REQUIREMENT leaf: `resolve/default-matches`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ConfigJSON = `{
  "default": "https://example.com",
  "domains": [
    {"server": "https://example.com", "token": "secret"}
  ]
}`
	return nil
}
```
