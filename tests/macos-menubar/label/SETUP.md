# Scenario

**Feature**: menu-bar title labels (FormatGrokLabel and FormatMenuBarLabel)

```
status/mode + usage fields -> label formatter -> compact menu-bar string
```

## Preconditions

Truncation budget comes from `menubar.TestExported_MaxLabelLen()` in `Run`.

## Steps

1. Leaf setup supplies formatter-specific inputs and `Op` when not `grok-label`.

## Context

Legacy grok-only leaves use default `grok-label` op; codex/rotating leaves set
`Op=menu-label`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// Leaves supply formatter-specific inputs; reset shared fields first.
	req.Op = ""
	req.Status = ""
	req.WeeklyLimit = ""
	req.ErrorMsg = ""
	req.DisplayMode = ""
	req.RotatingIndex = 0
	req.GrokStatus = ""
	req.GrokWeekly = ""
	req.GrokError = ""
	req.CodexStatus = ""
	req.CodexMonthly = ""
	req.CodexError = ""
	return nil
}
```