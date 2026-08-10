# Scenario

**Feature**: one event as NDJSON

```
# hub + --json
RunEventBusListen(JSON) -> NDJSON line with type/id
```

## Steps

1. Hub; one live event; MaxEvents=1; JSON=true.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	setListenHub(req)
	req.JSON = true
	seedOneLive(req)
	req.MaxEvents = 1
	return nil
}
```
