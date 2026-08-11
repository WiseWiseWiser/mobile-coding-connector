# Scenario

**Feature**: OpenTTY enabled path

```
opts.OpenTTY=true + OpenTTYSession inject -> open policy on printed events
```

## Steps

1. OpenTTY=true; InjectOpenHook=true; DialMode=inject.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	setListenInject(req)
	req.OpenTTY = true
	req.InjectOpenHook = true
	req.JSON = false
	return nil
}
```
