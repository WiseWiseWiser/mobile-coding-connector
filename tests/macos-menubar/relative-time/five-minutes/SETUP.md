# Scenario

**Feature**: grok reset 5m away → minutes only

```
FormatTimeLeft("July 6, 17:00 PT", now=Jul 6 16:55 PDT) -> "left 5m"
```

## Preconditions

Remaining duration is exactly 5 minutes.

## Steps

1. Set reset and fixed `now` per requirement scenario 6.

## Context

REQUIREMENT-DESIGN-menubar-display-v2.md: <1h → `left {m}m` only.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Reset = "July 6, 17:00 PT"
	req.NowRFC3339 = "2026-07-06T16:55:00-07:00"
	return nil
}
```