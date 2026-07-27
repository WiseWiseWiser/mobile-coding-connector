# Scenario

**Feature**: zero traffic after ready expires at idle

```
Start idle=5m -> Advance 5m1s -> Sweep -> session removed
```

## Steps

1. Idle=5m; AdvanceAfterReady=5m+1s; no traffic.

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
	req.Idle = 5 * time.Minute
	req.SendTraffic = false
	req.AdvanceAfterReady = 5*time.Minute + time.Second
	return nil
}
```
