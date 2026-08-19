# Scenario

**Feature**: concurrent ?refresh=1 callers share one Capture

```
N x GET ?refresh=1 -> one Capture in flight
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "inventory"
	req.ITermRunning = true
	req.WindowSpace = 1
	req.CoalesceN = 3
	req.CaptureHold = make(chan struct{})
	req.CaptureEntered = make(chan struct{}, 8)
	return nil
}
```
