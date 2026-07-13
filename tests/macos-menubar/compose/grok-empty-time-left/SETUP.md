# Scenario

**Feature**: grok compose-only ready line omits empty time_left

```
ComposeGrokDropdownLine(61%, "July 17, 08:55", "")
  -> "Grok: 61%(Weekly), Reset July 17, 08:55"
```

## Preconditions

`time_left` empty (unparseable reset on backend, or omitted).

## Steps

1. Set `Op=grok-compose`, ready, weekly=61%, reset_display, empty time_left.

## Context

REQUIREMENT scenario 6: no trailing `, left …` when time_left is empty.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "grok-compose"
	req.Status = "ready"
	req.WeeklyLimit = "61%"
	req.ResetDisplay = "July 17, 08:55"
	req.TimeLeft = ""
	return nil
}
```
