# Scenario

**Feature**: fixed grok display mode shows grok weekly limit

```
FormatMenuBarLabel(mode=grok, grok ready 6%) -> "Grok 6%"
```

## Preconditions

Display mode `grok` with grok usage ready.

## Steps

1. Set `Op=menu-label`, `DisplayMode=grok`, grok status fields.

## Context

REQUIREMENT leaf: `label/grok-fixed`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "menu-label"
	req.DisplayMode = "grok"
	req.GrokStatus = "ready"
	req.GrokWeekly = "6%"
	req.GrokError = ""
	req.CodexStatus = "ready"
	req.CodexMonthly = "58%"
	req.CodexError = ""
	return nil
}
```