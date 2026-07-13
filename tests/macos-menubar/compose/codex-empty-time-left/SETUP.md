# Scenario

**Feature**: codex compose-only ready line omits empty time_left

```
ComposeCodexDropdownLine(58%, 6,519/11,250, "Aug 1, 08:00", "")
  -> "Codex: 58%(Monthly) 6,519/11,250, Reset Aug 1, 08:00"
```

## Preconditions

`time_left` empty (unparseable reset on backend, or omitted).

## Steps

1. Set `Op=codex-compose`, ready, monthly/credits, reset_display, empty time_left.

## Context

REQUIREMENT scenario 6 (Codex analog).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "codex-compose"
	req.Status = "ready"
	req.MonthlyUsage = "58%"
	req.CreditsUsed = "6,519"
	req.CreditsTotal = "11,250"
	req.ResetDisplay = "Aug 1, 08:00"
	req.TimeLeft = ""
	return nil
}
```
