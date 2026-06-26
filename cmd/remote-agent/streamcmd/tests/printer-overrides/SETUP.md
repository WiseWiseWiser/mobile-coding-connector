# Scenario

**Feature**: Printer overrides replace A-path per type (B path)

```
# Printer.Log set -> custom formatter; other types use Print defaults
Printer.Log != nil overrides log only
```

## Preconditions

`Print: Logs` enabled so default log handler would otherwise run.

## Steps

Leaves configure specific `Printer` fields.

## Context

Requirement scenario `streamcmd-printer-override`.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/cmd/remote-agent/streamcmd"
)

func Setup(t *testing.T, req *Request) error {
	req.Print = streamcmd.Logs
	return nil
}
```
