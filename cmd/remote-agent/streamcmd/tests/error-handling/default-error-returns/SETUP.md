# Scenario

**Feature**: default error handler returns message

```
# nil Printer.Error -> errors.New(ev.Message)
{"type":"error","message":"doctor failed"} -> RunErr
```

## Preconditions

Zero `Printer` (all nil).

## Steps

1. Set `MockEvents` to error frame only.
2. Set `Print = streamcmd.Logs` (any flag).

## Context

Default B-path fallback for error type when `Printer.Error` is nil.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/cmd/remote-agent/streamcmd"
)

func Setup(t *testing.T, req *Request) error {
	req.Print = streamcmd.Logs
	req.MockEvents = []map[string]any{
		{"type": "error", "message": "doctor failed"},
	}
	return nil
}
```
