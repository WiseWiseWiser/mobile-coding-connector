# Scenario

**Feature**: grok reset 4h5m away → compound hours and minutes

```
FormatTimeLeft("July 6, 21:00 PT", now=Jul 6 16:55 PDT) -> "left 4h5m"
```

## Preconditions

Remaining duration is exactly 4 hours 5 minutes.

## Steps

1. Set reset and fixed `now` per requirement scenario 5.

## Context

REQUIREMENT-DESIGN-menubar-display-v2.md: 4h5m → `left 4h5m`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Reset = "July 6, 21:00 PT"
	req.NowRFC3339 = "2026-07-06T16:55:00-07:00"
	return nil
}
```