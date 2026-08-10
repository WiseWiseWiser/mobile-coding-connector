# Scenario

**Feature**: WS client receives event after HTTP POST /publish

```
# HTTP -> hub -> WS
POST loopback /publish -> Hub fan-out -> WS /api/event-bus/ws Event
```

## Steps

1. PublishVia=http; open publish (no token).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.PublishVia = "http"
	req.ServerToken = ""
	req.ListenAddr = "127.0.0.1:0"
	return nil
}
```
