# Scenario

**Feature**: codex usage service fetch via injectable in-process hook

```
TestExported_SetFetcher(mock) -> service FetchOnce -> CodexUsageResponse
```

## Preconditions

Service exposes `TestExported_SetFetcher` for deterministic snapshots.

## Steps

1. Set `Op=fetch` in leaves.
2. Leaf sets `FetchMode` to `success` or `error`.

## Context

Service-layer tests without shell `codex-show-status` binary.
`slow-boot-snapshot` uses `Op=fetch-inprocess` (real `agent/usage.Fetch`, no mock).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.Op == "" {
		req.Op = "fetch"
	}
	return nil
}
```