# Scenario

**Feature**: codex compose-only ready line with time_left

```
ComposeCodexDropdownLine(58%, 6,519/11,250, "Aug 1, 08:00", "left 26d")
  -> "Codex: 58%(Monthly) 6,519/11,250, Reset Aug 1, 08:00, left 26d"
```

## Preconditions

Structured fields already produced by backend (no raw next_reset parse).

## Steps

1. Set `Op=codex-compose`, ready, monthly/credits, reset_display, time_left.

## Context

REQUIREMENT scenario 5 (Codex analog of compose-only dropdown).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "codex-compose"
	req.Status = "ready"
	req.MonthlyUsage = "58%"
	req.CreditsUsed = "6,519"
	req.CreditsTotal = "11,250"
	req.ResetDisplay = "Aug 1, 08:00"
	req.TimeLeft = "left 26d"
	return nil
}
```
