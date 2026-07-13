# Scenario

**Feature**: second Get shortens time_left without re-fetch

```
# fixed reset_at = now1 + 4d2h
SeedReady(reset_at) -> Get@now1 => time_left "left 4d2h"
                    -> Get@now1+2h => time_left "left 4d"
```

## Preconditions

1. `ResetAtRFC3339=2026-07-17T10:55:00-07:00`.
2. First clock `2026-07-13T08:55:00-07:00` → exactly 4d2h remaining.
3. Second clock `2026-07-13T10:55:00-07:00` → exactly 4d remaining.
4. No PTY/mock fetch between Gets.

## Steps

1. Seed ready cache with fixed `reset_at` / `reset_display` / raw next_reset / weekly.
2. Call Get at now1 then now2 via harness `get-recompute` op.

## Context

REQUIREMENT scenario 2: countdown advances between PTY refreshes.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ResetAtRFC3339 = "2026-07-17T10:55:00-07:00"
	req.ResetDisplaySeed = "July 17, 10:55"
	req.NextResetSeed = "July 17, 10:55"
	req.WeeklyLimitSeed = "61%"
	req.NowRFC3339 = "2026-07-13T08:55:00-07:00"
	req.NowRFC3339Second = "2026-07-13T10:55:00-07:00"
	return nil
}
```
