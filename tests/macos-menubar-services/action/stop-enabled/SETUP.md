# Scenario

**Feature**: Stop enabled when process is alive

```
CanStopService(1234,...) -> true
```

## Preconditions

Service has a positive PID.

## Steps

1. Set `PID=1234`.

## Context

REQUIREMENT leaf: `action/stop-enabled`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.PID = 1234
	req.DesiredRunning = true
	return nil
}
```