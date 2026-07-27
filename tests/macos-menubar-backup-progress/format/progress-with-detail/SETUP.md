# Scenario

**Feature**: SSE progress frame with detail suffix

```
FormatBackupProgressFrame("home", "ok", "12 files") -> "[progress] home ok — 12 files"
```

## Preconditions

Non-empty detail appended after em dash separator.

## Steps

1. Op=format_progress with detail.

## Context

REQUIREMENT #10 (+ detail if present).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "format_progress"
	req.ProgressName = "home"
	req.ProgressStatus = "ok"
	req.ProgressDetail = "12 files"
	return nil
}
```
