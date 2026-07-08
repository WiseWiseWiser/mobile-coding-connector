# Scenario

**Feature**: grok reset 53h away → compound days and hours

```
FormatTimeLeft("July 8, 21:55 PT", now=Jul 6 16:55 PDT) -> "left 2d5h"
```

## Preconditions

Remaining duration is 53 hours (2 days 5 hours).

## Steps

1. Set reset and fixed `now` per requirement suggested leaf.

## Context

REQUIREMENT-DESIGN-menubar-display-v2.md scenario: 53h → `left 2d5h`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Reset = "July 8, 21:55 PT"
	req.NowRFC3339 = "2026-07-06T16:55:00-07:00"
	return nil
}
```