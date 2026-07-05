# Scenario

**Feature**: grok dropdown ready line

```
FormatGrokDropdownLine(6%, July 9, 16:55 PT) -> weekly limit + reset line
```

## Preconditions

Grok usage ready with weekly limit and reset time.

## Steps

1. Set `Op=grok-dropdown`, status=ready, weekly and reset fields.

## Context

REQUIREMENT leaf: `dropdown/grok-line`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "grok-dropdown"
	req.GrokStatus = "ready"
	req.WeeklyLimit = "6%"
	req.GrokReset = "July 9, 16:55 PT"
	return nil
}
```