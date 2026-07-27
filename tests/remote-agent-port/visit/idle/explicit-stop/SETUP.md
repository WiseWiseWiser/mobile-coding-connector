# Scenario

**Feature**: explicit stop cleans session and tunnel

```
Start -> Stop(id) -> list empty + provider Stop called
```

## Steps

1. Op=visit-stop; StopTarget empty → use session id.

```go
import (
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
	req.StopTarget = "" // id
	return nil
}
```
