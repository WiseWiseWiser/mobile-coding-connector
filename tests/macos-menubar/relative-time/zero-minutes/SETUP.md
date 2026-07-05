# Scenario

**Feature**: reset at or before now → zero minutes

```
FormatTimeLeft("July 6, 16:55 PT", now=Jul 6 16:55 PDT) -> "left 0min"
```

## Preconditions

Remaining duration is zero (reset time equals `now`).

## Steps

1. Set reset equal to `now`.

## Context

REQUIREMENT rule: duration ≤ 0 → `left 0min`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Reset = "July 6, 16:55 PT"
	req.NowRFC3339 = "2026-07-06T16:55:00-07:00"
	return nil
}
```