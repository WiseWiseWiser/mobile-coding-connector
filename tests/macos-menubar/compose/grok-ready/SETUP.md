# Scenario

**Feature**: grok compose-only ready line with time_left

```
ComposeGrokDropdownLine(61%, "July 17, 08:55", "left 4d")
  -> "Grok: 61%(Weekly), Reset July 17, 08:55, left 4d"
```

## Preconditions

Structured fields already produced by backend (no raw next_reset parse).

## Steps

1. Set `Op=grok-compose`, ready, weekly=61%, reset_display, time_left.

## Context

REQUIREMENT scenario 5: exact compose shape mirrors Swift concat.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "grok-compose"
	req.Status = "ready"
	req.WeeklyLimit = "61%"
	req.ResetDisplay = "July 17, 08:55"
	req.TimeLeft = "left 4d"
	return nil
}
```
