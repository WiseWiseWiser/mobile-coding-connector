# Scenario

**Feature**: codex dropdown shows full timeout error from daemon fetch

```
FormatCodexDropdownLine("error", ..., "timeout waiting for status output") -> full error line
```

## Preconditions

Dropdown row carries full daemon error while menu bar stays short (`Codex err`).

## Steps

1. Set `Op=codex-dropdown`, status=error, CodexError=timeout message from screenshot.

## Context

Documents menu-bar screenshot: `Codex: Error: timeout waiting for status output`.
Formatting is correct; fetch layer must not produce this error (see codex-usage leaves).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "codex-dropdown"
	req.CodexStatus = "error"
	req.CodexMonthly = ""
	req.CodexCreditsUsed = ""
	req.CodexCreditsTotal = ""
	req.CodexReset = ""
	req.CodexError = "timeout waiting for status output"
	return nil
}
```