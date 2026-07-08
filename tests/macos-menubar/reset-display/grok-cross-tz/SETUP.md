# Scenario

**Feature**: grok PT reset converted to EDT local wall clock

```
FormatResetDisplay("July 9, 17:55 PT", now=Jul 6 16:55 EDT) -> "July 9, 20:55"
```

## Preconditions

`now` is in `America/New_York` (EDT, UTC-4); PT is UTC-7 in July → +3h shift.

## Steps

1. Set grok reset and EDT `now` via RFC3339 offset `-04:00`.

## Context

REQUIREMENT scenario 7: `July 9, 17:55 PT` in EDT → `July 9, 20:55`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Reset = "July 9, 17:55 PT"
	req.NowRFC3339 = "2026-07-06T16:55:00-04:00"
	return nil
}
```