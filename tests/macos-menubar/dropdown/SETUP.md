# Scenario

**Feature**: dropdown single-line usage formatters

```
usage fields -> FormatGrokDropdownLine / FormatCodexDropdownLine -> dropdown text
```

## Preconditions

Ready-state dropdown lines per REQUIREMENT-DESIGN-codex-usage.md confirmed decisions.

## Steps

1. Leaf setup sets `Op` and formatter-specific inputs.

## Context

Dropdown leaves assert exact canonical v2 strings for grok weekly and codex monthly
usage: `Grok: {pct}(Weekly), Reset {local}, {timeLeft}` and
`Codex: {pct}(Monthly) {used}/{total}, Reset {local}, {timeLeft}`.
Unparseable reset omits `{timeLeft}`; error/loading lines unchanged.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.WeeklyLimit = ""
	req.GrokReset = ""
	req.CodexStatus = ""
	req.CodexMonthly = ""
	req.CodexCreditsUsed = ""
	req.CodexCreditsTotal = ""
	req.CodexReset = ""
	req.GrokError = ""
	req.CodexError = ""
	return nil
}
```