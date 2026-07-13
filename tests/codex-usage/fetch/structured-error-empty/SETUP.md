# Scenario

**Feature**: injected fetch failure → no invented structured reset fields

```
TestExported_SetFetcher(error) -> FetchOnce -> status=error
  reset_at, reset_display, time_left empty
```

## Preconditions

Injectable fetcher returns an error so the service records `status=error`.

## Steps

1. Set `FetchMode=error`.

## Context

REQUIREMENT-DESIGN-usage-structured-reset-ab.md scenario 3 (error path parity for
Codex): do not invent `reset_at` / `reset_display` / `time_left`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.FetchMode = "error"
	return nil
}
```
