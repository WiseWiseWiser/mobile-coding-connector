# Scenario

**Feature**: tunnel Start uses proxy port not app port

```
Start -> Provider.Start(proxyPort) where proxyPort != localPort
```

## Steps

1. Op=visit-proxy-hop; capture Start port.

```go
import (
	"testing"
	"time"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "visit-proxy-hop"
	req.Port = defaultTestPort
	req.Provider = "quick"
	enableOwnedQuick(req, true, true)
	req.Idle = 5 * time.Minute
	req.CaptureStartPort = true
	return nil
}
```
