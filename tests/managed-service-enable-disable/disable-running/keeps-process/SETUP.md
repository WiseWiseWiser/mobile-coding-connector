# Scenario

**Feature**: disable keeps a running process alive

```
# running enabled service
POST /api/services/disable?id=svc-run-001 -> message + enabled=false

# process must remain alive
GET /api/services -> pid > 0, processAlive(pid)
```

## Preconditions

Parent `disable-running` setup has started the service and issued disable.

## Steps

1. Inherit parent setup (enabled running service, disable action).
2. Assert API `message`, on-disk `enabled=false`, and live PID.

## Context

REQUIREMENT leaf: `disable-running/keeps-process`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.WaitAfterSecs = 0
	if req.Action != "disable" {
		req.Action = "disable"
	}
	req.UseBinary = true // L3 product-binary smoke
	return nil
}
```
