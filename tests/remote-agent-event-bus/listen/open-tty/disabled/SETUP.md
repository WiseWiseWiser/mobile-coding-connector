# Scenario

**Feature**: OpenTTY disabled (default)

```
opts.OpenTTY=false + inject hook installed
  -> OpenTTYSession never called even on agent.tty.started
```

## Steps

1. OpenTTY=false; InjectOpenHook=true so a mis-fire is observable.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	setListenInject(req)
	req.OpenTTY = false
	req.InjectOpenHook = true
	req.JSON = false
	return nil
}
```
