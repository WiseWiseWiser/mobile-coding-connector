# Scenario

**Feature**: rotating mode index 1 shows codex slot

```
FormatMenuBarLabel(mode=rotating, index=1, codex ready 58%) -> "Codex 58%"
```

## Preconditions

Rotating display with index 1 (codex slot).

## Steps

1. Set `Op=menu-label`, `DisplayMode=rotating`, `RotatingIndex=1`.

## Context

REQUIREMENT leaf: `label/rotating-codex-slot`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "menu-label"
	req.DisplayMode = "rotating"
	req.RotatingIndex = 1
	req.GrokStatus = "ready"
	req.GrokWeekly = "6%"
	req.GrokError = ""
	req.CodexStatus = "ready"
	req.CodexMonthly = "58%"
	req.CodexError = ""
	return nil
}
```