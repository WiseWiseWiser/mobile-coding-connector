# Scenario

**Feature**: HTTP through proxy resets last-activity

```
Start -> GET proxy -> LastActivity advances
```

## Steps

1. Op=visit-idle; SendTraffic=true; CaptureActivityBefore.

```go
import (
	"testing"
	"time"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "visit-idle"
	req.Port = defaultTestPort
	req.Provider = "quick"
	enableOwnedQuick(req, true, true)
	req.Idle = 10 * time.Minute
	req.SendTraffic = true
	req.CaptureActivityBefore = true
	// Do not advance past idle; only check activity reset.
	req.AdvanceAfterReady = 0
	return nil
}
```
