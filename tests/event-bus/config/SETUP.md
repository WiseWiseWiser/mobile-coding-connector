# Scenario

**Feature**: publish flag resolution and keep-alive argv forward

```
# pure helpers
DefaultPublishPort / ResolvePublishConfig / AppendEventBusServerArgs
```

## Preconditions

No network; pure functions on `server/eventbus`.

## Steps

1. Grouping sets Op resolve-config or append-args.
2. Leaves set PortFlag / TokenFlag / NoPublish / BaseArgs.

## Context

REQUIREMENT scenarios 7–8.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	// Op set by resolve/ or append-args/
	return nil
}
```
