# Scenario

**Feature**: streamcmd surfaces SSE error events

```
# default Error handler returns errors.New(message)
type=error -> Run returns err
```

## Preconditions

Mock events include `error` frame (no `done`).

## Steps

Child leaf sets error mock sequence.

## Context

Ensures CLI commands fail fast on fatal server errors.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/cmd/remote-agent/streamcmd"
)

func Setup(t *testing.T, req *Request) error {
	if req.Print == 0 {
		req.Print = streamcmd.Logs
	}
	return nil
}
```
