# Scenario

**Feature**: Get() recomputes time_left from cached reset_at (Codex)

```
SeedReady(reset_at) -> SetNow(t1) -> Get().time_left
                     -> SetNow(t2) -> Get().time_left (shorter, no re-fetch)
```

## Preconditions

1. Service exposes test hooks `TestExported_SeedReady` and `TestExported_SetNow`
   (implementer) so leaves can fix absolute reset and wall clock.
2. `Get()` recomputes `time_left` from cached `reset_at` + now without calling fetcher.

## Steps

1. Set `Op=get-recompute`.
2. Leaf seeds fixed `ResetAtRFC3339` and two `NowRFC3339*` clocks.

## Context

REQUIREMENT-DESIGN-usage-structured-reset-ab.md scenario 2 (Codex parity). Classic
TDD: RED until hooks + Get recompute exist.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "get-recompute"
	return nil
}
```
