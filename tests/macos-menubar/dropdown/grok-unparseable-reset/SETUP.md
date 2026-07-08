# Scenario

**Feature**: grok dropdown ready with unparseable reset omits relative suffix

```
FormatGrokDropdownLine(6%, soon) -> "Grok: 6%(Weekly), Reset soon" (no left)
```

## Preconditions

Reset string `soon` cannot be parsed for relative countdown.

## Steps

1. Set `Op=grok-dropdown`, status=ready, weekly=6%, reset=soon.

## Context

REQUIREMENT scenario 8: unparseable reset → show `Reset {raw}` only, no `left`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "grok-dropdown"
	req.GrokStatus = "ready"
	req.WeeklyLimit = "6%"
	req.GrokReset = "soon"
	req.NowRFC3339 = "2026-07-06T16:55:00-07:00"
	return nil
}
```