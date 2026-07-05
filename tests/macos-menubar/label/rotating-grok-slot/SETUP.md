# Scenario

**Feature**: rotating mode index 0 shows grok slot

```
FormatMenuBarLabel(mode=rotating, index=0, grok ready 6%) -> "Grok 6%"
```

## Preconditions

Rotating display with index 0 (grok slot).

## Steps

1. Set `Op=menu-label`, `DisplayMode=rotating`, `RotatingIndex=0`.

## Context

REQUIREMENT leaf: `label/rotating-grok-slot`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "menu-label"
	req.DisplayMode = "rotating"
	req.RotatingIndex = 0
	req.GrokStatus = "ready"
	req.GrokWeekly = "6%"
	req.GrokError = ""
	req.CodexStatus = "ready"
	req.CodexMonthly = "58%"
	req.CodexError = ""
	return nil
}
```