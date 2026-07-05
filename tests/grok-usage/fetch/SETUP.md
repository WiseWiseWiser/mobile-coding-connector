# Scenario

**Feature**: grok usage service fetch via injectable in-process hook

```
TestExported_SetFetcher(mock) -> service FetchOnce -> GrokUsageResponse
```

## Preconditions

Service exposes `TestExported_SetFetcher` for deterministic snapshots.

## Steps

1. Set `Op=fetch` in leaves.
2. Leaf sets `FetchMode` to `success` or `error`.

## Context

Service-layer tests without shell `grok-show-status` binary.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "fetch"
	return nil
}
```