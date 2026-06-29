# Scenario

**Feature**: custom Printer.Log writes to stderr with prefix

```
# B override: [custom] message on stderr; stdout has no default log line
Printer.Log -> fmt.Fprintf(os.Stderr, "[custom] %s\n", ...)
```

## Preconditions

Mock includes a `log` event.

## Steps

1. Set `Printer.Log` to custom stderr formatter.

## Context

Requirement scenario 6 — B override wins for `log` only.

```go
import (
	"fmt"
	"os"
	"testing"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/ai-critic/cmd/agentcli/streamcmd"
)

func Setup(t *testing.T, req *Request) error {
	req.Printer = streamcmd.Printer{
		Log: func(ev client.StreamEvent) error {
			_, err := fmt.Fprintf(os.Stderr, "[custom] %s\n", ev.Message)
			return err
		},
	}
	return nil
}
```
