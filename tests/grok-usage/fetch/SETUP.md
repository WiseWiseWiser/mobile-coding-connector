# Scenario

**Feature**: grok usage service fetch via injectable HTTP fetcher

```
FetchMode mock -> service FetchOnce -> GrokUsageResponse
```

## Preconditions

`TestExported_SetFetcher` injects HTTP-shaped `FetchResult` (no PTY).

## Steps

1. Set `Op=fetch` in leaves.

## Context

Service-layer tests without full daemon HTTP.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "fetch"
	return nil
}
```
