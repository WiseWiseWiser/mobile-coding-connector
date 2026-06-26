# Scenario

**Feature**: Print flags select builtin formatters (A path)

```
# Print flags enable default log/section/progress printers
Print: Logs|Sections|ProgressChecks -> builtin fmt lines
```

## Preconditions

`Printer` fields are nil (zero value).

## Steps

1. Set `Print = Logs | Sections | ProgressChecks`.

## Context

Requirement scenario `streamcmd-default-printers`.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/cmd/remote-agent/streamcmd"
)

func Setup(t *testing.T, req *Request) error {
	req.Print = streamcmd.Logs | streamcmd.Sections | streamcmd.ProgressChecks
	return nil
}
```
