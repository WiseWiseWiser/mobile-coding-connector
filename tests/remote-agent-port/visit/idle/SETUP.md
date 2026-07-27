# Scenario

**Feature**: reverse-proxy hop and idle shutdown

```
Start -> proxy hop -> optional traffic -> Advance clock -> Sweep -> session gone or alive
```

## Preconditions

Fake clock via SetNow; short Idle on sessions; fake Provider ready immediately.

## Steps

1. Leaf configures Idle, AdvanceAfterReady, SendTraffic.
2. Op=visit-idle or visit-proxy-hop or visit-stop.
3. Assert hop port, activity, alive flag, stop counts.

## Context

Idle clock starts at tunnel-ready; zero traffic still expires.

```go
import (
	"testing"
	"time"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Port = defaultTestPort
	enableOwnedQuick(req, true, true)
	req.Provider = "quick"
	req.Idle = 5 * time.Minute
	return nil
}
```
