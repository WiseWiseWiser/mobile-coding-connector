# Scenario

**Feature**: WS client receives Hub.Publish

```
# direct hub path
WS dial /api/event-bus/ws -> Hub.Publish(ev) -> WS frame Event
```

## Steps

1. PublishVia=hub.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.PublishVia = "hub"
	return nil
}
```
