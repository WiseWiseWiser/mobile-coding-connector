# Scenario

**Feature**: grok reset 76h away → compound days and hours

```
FormatTimeLeft("July 9, 20:55 PT", now=Jul 6 16:55 PDT) -> "left 3d4h"
```

## Preconditions

Remaining duration is 76 hours (3 days 4 hours).

## Steps

1. Set reset and fixed `now` per requirement scenario 3.

## Context

REQUIREMENT-DESIGN-menubar-display-v2.md: ≥24h uses `d`+`h` compound; omit zero-hour tail.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Reset = "July 9, 20:55 PT"
	req.NowRFC3339 = "2026-07-06T16:55:00-07:00"
	return nil
}
```