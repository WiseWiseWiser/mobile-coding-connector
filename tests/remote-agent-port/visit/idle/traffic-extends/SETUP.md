# Scenario

**Feature**: traffic near idle deadline extends lifetime

```
Start idle=5m -> traffic -> Advance 5m from traffic -> still alive; then more -> expires optional
```

## Steps

1. SendTraffic; AdvanceAfterTraffic = 5m - 1s (still within new idle window from traffic).
2. Actually: AdvanceAfterReady is used after traffic in Run when AdvanceAfterTraffic set.
3. Set AdvanceAfterTraffic = 4*time.Minute (within 5m idle after traffic).

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
	req.SendTraffic = true
	// After traffic, advance less than idle → still alive
	req.AdvanceAfterTraffic = 4 * time.Minute
	req.AdvanceAfterReady = 0
	return nil
}
```
