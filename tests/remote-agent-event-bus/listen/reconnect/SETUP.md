# Scenario

**Feature**: disconnect warning + reconnect

```
# injectable DialWS drops once
listen -> event1 -> disconnect -> warning: -> redial -> event2
```

## Steps

1. DialMode=drop-once; inject two events.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.DialMode = "drop-once"
	return nil
}
```
