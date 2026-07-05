# Scenario

**Bug**: menu bar must show short fixed codex error label

```
formatCodexLabel("error","","fork/exec ...") -> "Codex err"
```

## Preconditions

Codex usage fetch fails with exec error; menu bar hides full path.

## Steps

1. Use `Op=menu-label`, `DisplayMode=codex`, codex status=error.

## Context

REQUIREMENT leaf: `label/codex-error`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "menu-label"
	req.DisplayMode = "codex"
	req.CodexStatus = "error"
	req.CodexMonthly = ""
	req.CodexError = "fork/exec /Users/xhd2015/go/bin/codex-show-status: no such file or directory"
	return nil
}
```