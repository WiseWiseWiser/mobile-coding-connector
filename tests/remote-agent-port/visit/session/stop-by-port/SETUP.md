# Scenario

**Feature**: stop visit by local port

```
Start -> Stop("18080") -> empty list
```

## Steps

1. Op=visit-stop; StopTarget = port string.

```go
import (
	"strconv"
	"testing"
	"time"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "visit-stop"
	req.Port = defaultTestPort
	req.Provider = "quick"
	enableOwnedQuick(req, true, true)
	req.Idle = 10 * time.Minute
	req.StopTarget = strconv.Itoa(defaultTestPort)
	return nil
}
```
