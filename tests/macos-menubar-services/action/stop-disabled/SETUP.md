# Scenario

**Feature**: Stop disabled when nothing to stop

```
CanStopService(0,false) -> false
```

## Preconditions

No live PID and daemon is not trying to keep the service running.

## Steps

1. Set `PID=0`, `DesiredRunning=false`.

## Context

REQUIREMENT leaf: `action/stop-disabled`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.PID = 0
	req.DesiredRunning = false
	return nil
}
```