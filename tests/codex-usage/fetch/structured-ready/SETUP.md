# Scenario

**Feature**: injected fetch success → structured reset_at / reset_display / time_left

```
# injectable snapshot Reset=08:00 on 1 Aug → structured A+B fields
TestExported_SetFetcher(success) -> FetchOnce -> CodexUsageResponse
  status=ready, next_reset raw, reset_at RFC3339, reset_display, time_left
```

## Preconditions

1. Injectable success snapshot returns monthly 58%, credits, `Reset=08:00 on 1 Aug`.
2. Service on success must set A+B structured fields (not only raw `next_reset`).

## Steps

1. Set `FetchMode=success`.

## Context

REQUIREMENT-DESIGN-usage-structured-reset-ab.md scenario 4 (Codex fetch success
structured fields). Classic TDD: RED until service populates `ResetAt`,
`ResetDisplay`, and `TimeLeft`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.FetchMode = "success"
	return nil
}
```
