# Scenario

**Feature**: SSE progress frame without detail

```
FormatBackupProgressFrame("home", "ok", "") -> "[progress] home ok"
```

## Preconditions

SSE `type=progress` with name + status; empty detail.

## Steps

1. Op=format_progress; name+status; empty detail.

## Context

REQUIREMENT #10.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "format_progress"
	req.ProgressName = "home"
	req.ProgressStatus = "ok"
	req.ProgressDetail = ""
	return nil
}
```
