# Scenario

**Feature**: grok reset 3h5m away → compound hours and minutes

```
FormatTimeLeft("July 6, 20:00 PT", now=Jul 6 16:55 PDT) -> "left 3h5m"
```

## Preconditions

Remaining duration is 3 hours 5 minutes.

## Steps

1. Set reset and fixed `now`.

## Context

REQUIREMENT-DESIGN-menubar-display-v2.md: ≥1h and <24h uses `h`+`m` compound.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Reset = "July 6, 20:00 PT"
	req.NowRFC3339 = "2026-07-06T16:55:00-07:00"
	return nil
}
```