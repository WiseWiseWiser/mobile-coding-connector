# Scenario

**Feature**: PruneBackupFiles retention (today 10 / 1 per day / drop older)

```
entries + now -> keep set + delete set
# today: newest 10; past days within 7 calendar days: 1 each; older: none
```

## Preconditions

`Op=retention`. `now` fixed at `2026-07-10T15:00:00Z` (UTC calendar day = 2026-07-10).
Defaults: keepTodayN=10, dailyDays=7.

## Steps

1. Leaf supplies EntriesJSON with synthetic paths and mod times.

## Context

REQUIREMENT retention scenarios 17–20.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "retention"
	if req.NowRFC3339 == "" {
		req.NowRFC3339 = "2026-07-10T15:00:00Z"
	}
	return nil
}
```
