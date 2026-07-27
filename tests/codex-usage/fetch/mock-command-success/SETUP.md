# Scenario

**Feature**: injected fetch success → service ready

```
injectable success snapshot -> status ready + usage fields
```

## Preconditions

Fetcher returns canonical codex usage snapshot (58%, credits, reset).

## Steps

1. `FetchMode=success`.

## Context

REQUIREMENT leaf: `fetch/mock-command-success`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.FetchMode = "success"
	return nil
}
```