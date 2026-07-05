# Scenario

**Feature**: sub-minute remainder floors to at least 1 minute

```
FormatTimeLeft("July 6, 16:56:30 PT", now=Jul 6 16:55 PDT) -> "left 1min"
```

## Preconditions

Remaining duration is 90 seconds (1.5 minutes) — floors to `left 1min`.

## Steps

1. Set reset with seconds and fixed `now`.

## Context

REQUIREMENT rule: minutes floor to at least 1 when 0 < duration < 1h.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Reset = "July 6, 16:56:30 PT"
	req.NowRFC3339 = "2026-07-06T16:55:00-07:00"
	return nil
}
```