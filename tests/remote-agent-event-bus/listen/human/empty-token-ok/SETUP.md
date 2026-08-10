# Scenario

**Feature**: empty token still listens

```
# token optional
RunEventBusListen(Token="") -> connected + one event
```

## Steps

1. EmptyToken=true; hub + one live event.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	setListenHub(req)
	req.EmptyToken = true
	req.Token = "should-be-cleared"
	seedOneLive(req)
	req.MaxEvents = 1
	return nil
}
```
