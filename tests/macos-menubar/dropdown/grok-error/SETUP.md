# Scenario

**Feature**: grok dropdown shows full error message

```
FormatGrokDropdownLine("error", "", "", "timeout waiting") -> "Grok: Error: timeout waiting"
```

## Preconditions

Dropdown row carries full daemon error while menu bar stays short.

## Steps

1. Set `Op=grok-dropdown`, status=error, GrokError=timeout message.

## Context

REQUIREMENT leaf: `dropdown/grok-error`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "grok-dropdown"
	req.GrokStatus = "error"
	req.WeeklyLimit = ""
	req.GrokReset = ""
	req.GrokError = "timeout waiting"
	return nil
}
```