# Scenario

**Feature**: codex dropdown shows full fork/exec error

```
FormatCodexDropdownLine("error", ..., fork/exec msg) -> "Codex: Error: {full msg}"
```

## Preconditions

Typical bundled-install failure: missing `codex-show-status` beside `ai-critic`.

## Steps

1. Set `Op=codex-dropdown`, status=error, full exec error in `CodexError`.

## Context

REQUIREMENT leaf: `dropdown/codex-error`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "codex-dropdown"
	req.CodexStatus = "error"
	req.CodexMonthly = ""
	req.CodexCreditsUsed = ""
	req.CodexCreditsTotal = ""
	req.CodexReset = ""
	req.CodexError = "fork/exec /Users/xhd2015/go/bin/codex-show-status: no such file or directory"
	return nil
}
```