# Scenario

**Feature**: fixed codex display mode shows codex monthly usage

```
FormatMenuBarLabel(mode=codex, codex ready 58%) -> "Codex 58%"
```

## Preconditions

Display mode `codex` with codex usage ready.

## Steps

1. Set `Op=menu-label`, `DisplayMode=codex`, codex status fields.

## Context

REQUIREMENT leaf: `label/codex-fixed`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "menu-label"
	req.DisplayMode = "codex"
	req.GrokStatus = "ready"
	req.GrokWeekly = "6%"
	req.GrokError = ""
	req.CodexStatus = "ready"
	req.CodexMonthly = "58%"
	req.CodexError = ""
	return nil
}
```