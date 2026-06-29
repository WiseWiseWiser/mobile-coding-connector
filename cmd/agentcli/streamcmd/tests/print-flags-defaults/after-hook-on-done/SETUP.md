# Scenario

**Feature**: After hook runs after done event

```
# stream completes -> Printer.Done (optional) -> After(done map)
done payload available to After hook
```

## Preconditions

`Print` includes at least one flag so stream runs to completion.

## Steps

1. Set `Print = streamcmd.Logs`.
2. Set `After` to a no-op (root wraps to record invocation).

## Context

Doctor command uses `After` for local client checks after server stream ends.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/cmd/agentcli/streamcmd"
)

func Setup(t *testing.T, req *Request) error {
	req.Print = streamcmd.Logs
	return nil
}
```
