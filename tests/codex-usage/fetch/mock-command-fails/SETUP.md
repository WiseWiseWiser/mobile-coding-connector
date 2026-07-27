# Scenario

**Feature**: injected fetch failure → service error status

```
injectable fetch error -> status error
```

## Preconditions

Fetcher returns a non-nil error (no subprocess exec).

## Steps

1. `FetchMode=error`.

## Context

REQUIREMENT leaf: `fetch/mock-command-fails`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.FetchMode = "error"
	return nil
}
```