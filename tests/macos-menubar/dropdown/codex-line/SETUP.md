# Scenario

**Feature**: codex dropdown ready line

```
FormatCodexDropdownLine(58%, 6,519, 11,250, reset) -> monthly usage + credits line
```

## Preconditions

Codex usage ready with monthly usage, credits, and reset time.

## Steps

1. Set `Op=codex-dropdown`, status=ready, monthly/credits/reset fields.

## Context

REQUIREMENT leaf: `dropdown/codex-line`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "codex-dropdown"
	req.CodexStatus = "ready"
	req.CodexMonthly = "58%"
	req.CodexCreditsUsed = "6,519"
	req.CodexCreditsTotal = "11,250"
	req.CodexReset = "08:00 on 1 Aug"
	return nil
}
```