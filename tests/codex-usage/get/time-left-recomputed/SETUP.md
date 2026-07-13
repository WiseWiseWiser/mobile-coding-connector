# Scenario

**Feature**: second Get shortens Codex time_left without re-fetch

```
# fixed reset_at = now1 + 4d2h
SeedReady(reset_at) -> Get@now1 => time_left "left 4d2h"
                    -> Get@now1+2h => time_left "left 4d"
```

## Preconditions

1. `ResetAtRFC3339=2026-08-01T10:00:00-07:00` (absolute).
2. First clock `2026-07-28T08:00:00-07:00` → exactly 4d2h remaining.
3. Second clock `2026-07-28T10:00:00-07:00` → exactly 4d remaining.
4. No injectable re-fetch between Gets.

## Steps

1. Seed ready cache with fixed `reset_at` / `reset_display` / raw next_reset / monthly.
2. Call Get at now1 then now2 via harness `get-recompute` op.

## Context

REQUIREMENT scenario 2 parity for Codex service Get recompute.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ResetAtRFC3339 = "2026-08-01T10:00:00-07:00"
	req.ResetDisplaySeed = "Aug 1, 10:00"
	req.NextResetSeed = "10:00 on 1 Aug"
	req.MonthlyUsageSeed = "58%"
	req.NowRFC3339 = "2026-07-28T08:00:00-07:00"
	req.NowRFC3339Second = "2026-07-28T10:00:00-07:00"
	return nil
}
```
